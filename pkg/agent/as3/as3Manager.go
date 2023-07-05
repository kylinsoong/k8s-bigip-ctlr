/*-
 * Copyright (c) 2016-2021, F5 Networks, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package as3

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/F5Networks/k8s-bigip-ctlr/v2/pkg/writer"

	. "github.com/F5Networks/k8s-bigip-ctlr/v2/pkg/resource"
	log "github.com/F5Networks/k8s-bigip-ctlr/v2/pkg/vlogger"
)

const (
	svcTenantLabel      = "cis.f5.com/as3-tenant="
	svcAppLabel         = "cis.f5.com/as3-app="
	svcPoolLabel        = "cis.f5.com/as3-pool="
	as3SupportedVersion = 3.18
	// Update as3Version,defaultAS3Version,defaultAS3Build while updating AS3 validation schema.
	// While upgrading version update $id value in schema json to https://raw.githubusercontent.com/F5Networks/f5-appsvcs-extension/master/schema/latest/as3-schema.json
	as3Version           = 3.45
	defaultAS3Version    = "3.45.0"
	defaultAS3Build      = "5"
	as3tenant            = "Tenant"
	as3class             = "class"
	as3SharedApplication = "Shared"
	as3application       = "Application"
	as3shared            = "shared"
	as3template          = "template"
	// as3SchemaLatestURL   = "https://raw.githubusercontent.com/F5Networks/f5-appsvcs-extension/master/schema/latest/as3-schema.json"
	as3defaultRouteDomain = "defaultRouteDomain"
	as3SchemaFileName     = "as3-schema-3.45.0-5-cis.json"
)

var baseAS3Config = `{
	"$schema": "https://raw.githubusercontent.com/F5Networks/f5-appsvcs-extension/master/schema/%s/as3-schema-%s.json",
	"class": "AS3",
	"declaration": {
	  "class": "ADC",
	  "schemaVersion": "%s",
	  "id": "urn:uuid:85626792-9ee7-46bb-8fc8-4ba708cfdc1d",
	  "label": "CIS Declaration",
	  "remark": "Auto-generated by CIS",
	  "controls": {
		 "class": "Controls",
		 "userAgent": "CIS Configured AS3"
	  }
	}
  }
  `

// AS3Config consists of all the AS3 related configurations
type AS3Config struct {
	resourceConfig        as3ADC
	configmaps            []*AS3ConfigMap
	overrideConfigmapData string
	tenantMap             map[string]interface{}
	unifiedDeclaration    as3Declaration
}

// ActiveAS3ConfigMap user defined ConfigMap for global availability.
type AS3ConfigMap struct {
	Name      string   // AS3 specific ConfigMap name
	Namespace string   // AS3 specific ConfigMap namespace
	config    as3ADC   // if AS3 Name is present, populate this with AS3 template data.
	endPoints []Member // Endpoints of all the pools in the configmap
	Validated bool     // Json Schema validated ok
}

// AS3Manager holds all the AS3 orchestration specific config
type AS3Manager struct {
	as3Validation             bool
	as3Validated              map[string]bool
	sslInsecure               bool
	enableTLS                 string
	tls13CipherGroupReference string
	ciphers                   string
	// Active User Defined ConfigMap details
	as3ActiveConfig AS3Config
	As3SchemaLatest string
	// Override existing as3 declaration with this configmap
	OverriderCfgMapName string
	// Path of schemas reside locally
	SchemaLocalPath string
	// POSTs configuration to BIG-IP using AS3
	PostManager *PostManager
	// To put list of tenants in BIG-IP REST call URL that are in AS3 declaration
	FilterTenants bool
	failedContext failureContext
	ReqChan       chan MessageRequest
	RspChan       chan interface{}
	userAgent     string
	l2l3Agent     L2L3Agent
	ResourceRequest
	ResourceResponse
	as3Version                string
	as3SchemaVersion          string
	as3Release                string
	unprocessableEntityStatus bool
	shareNodes                bool
	defaultRouteDomain        int
	poolMemberType            string
	bigIPAS3Version           float64
	as3LogLevel               *string
	as3DeclarationPersistence *bool
}

// Struct to allow NewManager to receive all or only specific parameters.
type Params struct {
	// Package local for unit testing only
	SchemaLocal               string
	AS3Validation             bool
	SSLInsecure               bool
	IPAM                      bool
	EnableTLS                 string
	TLS13CipherGroupReference string
	Ciphers                   string
	// Agent                     string
	OverriderCfgMapName string
	SchemaLocalPath     string
	FilterTenants       bool
	BIGIPUsername       string
	BIGIPPassword       string
	BIGIPURL            string
	TrustedCerts        string
	AS3PostDelay        int
	ConfigWriter        writer.Writer
	EventChan           chan interface{}
	// Log the AS3 response body in Controller logs
	LogResponse               bool
	ShareNodes                bool
	RspChan                   chan interface{}
	UserAgent                 string
	As3Version                string
	As3Release                string
	As3SchemaVersion          string
	unprocessableEntityStatus bool
	DefaultRouteDomain        int
	PoolMemberType            string
	HTTPClientMetrics         bool
}

type failureContext struct {
	failedTenants map[string]as3Declaration
}

// Create and return a new app manager that meets the Manager interface
func NewAS3Manager(params *Params) *AS3Manager {
	as3Manager := AS3Manager{
		as3Validation:             params.AS3Validation,
		as3Validated:              map[string]bool{},
		sslInsecure:               params.SSLInsecure,
		enableTLS:                 params.EnableTLS,
		tls13CipherGroupReference: params.TLS13CipherGroupReference,
		ciphers:                   params.Ciphers,
		SchemaLocalPath:           params.SchemaLocal,
		FilterTenants:             params.FilterTenants,
		failedContext:             failureContext{failedTenants: make(map[string]as3Declaration)},
		RspChan:                   params.RspChan,
		userAgent:                 params.UserAgent,
		as3Version:                params.As3Version,
		as3Release:                params.As3Release,
		as3SchemaVersion:          params.As3SchemaVersion,
		OverriderCfgMapName:       params.OverriderCfgMapName,
		shareNodes:                params.ShareNodes,
		defaultRouteDomain:        params.DefaultRouteDomain,
		poolMemberType:            params.PoolMemberType,
		as3ActiveConfig:           AS3Config{tenantMap: make(map[string]interface{})},
		l2l3Agent: L2L3Agent{
			eventChan:    params.EventChan,
			configWriter: params.ConfigWriter,
		},
		PostManager: NewPostManager(PostParams{
			BIGIPUsername:     params.BIGIPUsername,
			BIGIPPassword:     params.BIGIPPassword,
			BIGIPURL:          params.BIGIPURL,
			TrustedCerts:      params.TrustedCerts,
			SSLInsecure:       params.SSLInsecure,
			AS3PostDelay:      params.AS3PostDelay,
			LogResponse:       params.LogResponse,
			HTTPClientMetrics: params.HTTPClientMetrics,
		}),
	}

	if as3Manager.tls13CipherGroupReference == "" {
		as3Manager.tls13CipherGroupReference = "/Common/f5-default"
	}

	as3Manager.fetchAS3Schema()

	return &as3Manager
}

func updateTenantMap(tempAS3Config AS3Config) AS3Config {
	// Parse as3Config.configmaps , extract all tenants and store in tenantMap.
	for _, cm := range tempAS3Config.configmaps {
		for tenantName, tenant := range cm.config {
			tempAS3Config.tenantMap[tenantName] = tenant
		}
	}
	return tempAS3Config
}

func (am *AS3Manager) postAS3Declaration(rsReq ResourceRequest) (bool, string) {
	am.ResourceRequest = rsReq

	// as3Config := am.as3ActiveConfig
	as3Config := &AS3Config{
		tenantMap: make(map[string]interface{}),
	}

	// Process Route or Ingress
	as3Config.resourceConfig = am.prepareAS3ResourceConfig()

	// Process all Configmaps (including overrideAS3)
	as3Config.configmaps, as3Config.overrideConfigmapData = am.prepareResourceAS3ConfigMaps()

	if am.FilterTenants {
		updateTenantMap(*as3Config)
	}

	return am.postAS3Config(*as3Config)
}

func (am *AS3Manager) getADC() map[string]interface{} {
	var as3Obj map[string]interface{}

	baseAS3ConfigTemplate := fmt.Sprintf(baseAS3Config, am.as3Version, am.as3Release, am.as3SchemaVersion)
	_ = json.Unmarshal([]byte(baseAS3ConfigTemplate), &as3Obj)

	return as3Obj
}

func (am *AS3Manager) prepareTenantDeclaration(cfg *AS3Config, tenantName string) as3Declaration {
	as3Obj := am.getADC()
	adc, _ := as3Obj["declaration"].(map[string]interface{})

	adc[tenantName] = cfg.tenantMap[tenantName]

	unifiedDecl, err := json.Marshal(as3Obj)
	if err != nil {
		log.Debugf("[AS3] Unified declaration: %v\n", err)
	}

	return as3Declaration(unifiedDecl)
}

func (am *AS3Manager) processResponseCode(responseCode string, partition string, decl as3Declaration) {
	if responseCode != responseStatusOk && responseCode != responseStatusUnprocessableEntity {
		am.failedContext.failedTenants[partition] = decl
	} else {
		am.excludePartitionFromFailureTenantList(partition)
	}
}

func (am *AS3Manager) excludePartitionFromFailureTenantList(partition string) {
	for tenant := range am.failedContext.failedTenants {
		if tenant == partition {
			delete(am.failedContext.failedTenants, partition)
		}
	}
}

func (am *AS3Manager) processTenantDeletion(tempAS3Config AS3Config) (bool, string) {
	// Delete Tenants from as3ActiveConfig.tenantMap
	deletedTenants := am.getDeletedTenantsFromTenantMap(tempAS3Config.tenantMap)
	responseStatusList := getResponseStatusList()

	if len(deletedTenants) > 0 {
		for _, partition := range deletedTenants {

			// Update as3ActiveConfig
			delete(am.as3ActiveConfig.tenantMap, partition)

			_, responseCode := am.DeleteAS3Tenant(partition)
			responseStatusList[responseCode] = responseStatusList[responseCode] + 1

			am.processResponseCode(responseCode, partition, am.getEmptyAs3Declaration(partition))

		}

		resBool, resCode := processResponseCodeList(responseStatusList)
		// Update as3ActiveConfig
		if resCode == responseStatusOk {
			am.as3ActiveConfig.updateConfig(tempAS3Config)
		}
		return resBool, resCode
	}
	return true, responseStatusDummy
}

func getResponseStatusList() map[string]int {
	responseStatusList := map[string]int{responseStatusNotFound: 0, responseStatusServiceUnavailable: 0, responseStatusOk: 0, responseStatusCommon: 0, responseStatusUnprocessableEntity: 0, responseStatusDummy: 0}
	return responseStatusList
}

func (am *AS3Manager) processFilterTenants(tempAS3Config AS3Config) (bool, string) {
	// Delete Tenants from as3ActiveConfig.tenantMap
	_, deleteResponseCode := am.processTenantDeletion(tempAS3Config)

	responseStatusList := getResponseStatusList()
	for partition, tenant := range tempAS3Config.tenantMap {
		if tempAS3Config.tenantIsValid(partition) && !reflect.DeepEqual(am.as3ActiveConfig.tenantMap[partition], tenant) {
			tenantDecl := am.prepareTenantDeclaration(&tempAS3Config, partition)
			// Update as3ActiveConfig
			am.as3ActiveConfig.tenantMap[partition] = tempAS3Config.tenantMap[partition]
			am.as3ActiveConfig.updateConfig(tempAS3Config)

			log.Debugf("[AS3] Posting AS3 Declaration")
			_, responseCode := am.PostManager.postConfigRequests(string(tenantDecl), am.PostManager.getAS3APIURL([]string{partition}))
			responseStatusList[responseCode] = responseStatusList[responseCode] + 1

			am.processResponseCode(responseCode, partition, tenantDecl)
		}
	}
	responseStatusList[deleteResponseCode] = responseStatusList[deleteResponseCode] + 1
	return processResponseCodeList(responseStatusList)
}

func processResponseCodeList(responseList map[string]int) (bool, string) {
	if responseList[responseStatusServiceUnavailable] > 0 {
		return false, responseStatusServiceUnavailable
	}
	if responseList[responseStatusNotFound] > 0 {
		return true, responseStatusNotFound
	}
	if responseList[responseStatusCommon] > 0 {
		return false, responseStatusCommon
	}
	if responseList[responseStatusDummy] > 0 {
		return true, responseStatusDummy
	}
	if responseList[responseStatusUnprocessableEntity] > 0 {
		return true, responseStatusUnprocessableEntity
	}
	if responseList[responseStatusOk] > 0 {
		return true, responseStatusOk
	}
	return false, responseStatusCommon
}

func (am *AS3Manager) postAS3Config(tempAS3Config AS3Config) (bool, string) {
	if am.FilterTenants {
		return am.processFilterTenants(tempAS3Config)
	}
	unifiedDecl := am.getUnifiedDeclaration(&tempAS3Config)
	if unifiedDecl == "" {
		return true, ""
	}
	if DeepEqualJSON(am.as3ActiveConfig.unifiedDeclaration, unifiedDecl) {
		return !am.unprocessableEntityStatus, ""
	}

	if am.as3Validation == true {
		if ok := am.validateAS3Template(string(unifiedDecl)); !ok {
			return true, ""
		}
	}

	log.Debugf("[AS3] Posting AS3 Declaration")

	am.as3ActiveConfig.updateConfig(tempAS3Config)

	return am.PostManager.postConfigRequests(string(unifiedDecl), am.PostManager.getAS3APIURL(nil))
}

func (cfg *AS3Config) updateConfig(newAS3Cfg AS3Config) {
	cfg.resourceConfig = newAS3Cfg.resourceConfig
	cfg.unifiedDeclaration = newAS3Cfg.unifiedDeclaration
	cfg.configmaps = newAS3Cfg.configmaps
	cfg.overrideConfigmapData = newAS3Cfg.overrideConfigmapData
}

func (cfg *AS3Config) tenantIsValid(tenant string) bool {
	for _, cfg := range cfg.configmaps {
		for t := range cfg.config {
			if t == tenant {
				return cfg.Validated
			}
		}
	}
	return false
}

func (am *AS3Manager) getUnifiedDeclaration(cfg *AS3Config) as3Declaration {
	// Need to process Routes
	var as3Obj map[string]interface{}

	baseAS3ConfigTemplate := fmt.Sprintf(baseAS3Config, am.as3Version, am.as3Release, am.as3SchemaVersion)
	_ = json.Unmarshal([]byte(baseAS3ConfigTemplate), &as3Obj)
	adc, _ := as3Obj["declaration"].(map[string]interface{})
	for tenantName, tenant := range cfg.resourceConfig {
		adc[tenantName] = tenant
	}
	for _, cm := range cfg.configmaps {
		for tenantName, tenant := range cm.config {
			adc[tenantName] = tenant
		}
	}
	for _, tnt := range am.getDeletedTenants(adc) {
		// This config deletes the partition in BIG-IP
		adc[tnt] = as3Tenant{
			"class": "Tenant",
		}
	}
	// Update AS3 logLevel parameter if specified
	if am.as3LogLevel != nil {
		as3Obj["logLevel"] = *am.as3LogLevel
	}
	// Update AS3 persist parameter if specified
	if am.as3DeclarationPersistence != nil {
		as3Obj["persist"] = *am.as3DeclarationPersistence
	}
	unifiedDecl, err := json.Marshal(as3Obj)
	if err != nil {
		log.Debugf("[AS3] Unified declaration: %v\n", err)
	}

	if cfg.overrideConfigmapData == "" {
		cfg.unifiedDeclaration = as3Declaration(unifiedDecl)
		return as3Declaration(unifiedDecl)
	}

	overriddenUnifiedDecl := ValidateAndOverrideAS3JsonData(
		cfg.overrideConfigmapData,
		string(unifiedDecl),
	)
	if overriddenUnifiedDecl == "" {
		log.Debug("[AS3] Failed to override AS3 Declaration")
		cfg.unifiedDeclaration = as3Declaration(unifiedDecl)
		return as3Declaration(unifiedDecl)
	}
	cfg.unifiedDeclaration = as3Declaration(overriddenUnifiedDecl)
	return as3Declaration(overriddenUnifiedDecl)
}

// Function to prepare empty AS3 declaration
func (am *AS3Manager) getEmptyAs3Declaration(partition string) as3Declaration {
	var as3Config map[string]interface{}
	baseAS3ConfigEmpty := fmt.Sprintf(baseAS3Config, am.as3Version, am.as3Release, am.as3SchemaVersion)
	_ = json.Unmarshal([]byte(baseAS3ConfigEmpty), &as3Config)
	decl := as3Config["declaration"].(map[string]interface{})

	controlObj := make(as3Control)
	controlObj.initDefault(am.userAgent)
	decl["controls"] = controlObj
	if partition != "" {
		decl[partition] = map[string]string{"class": "Tenant"}
	}
	data, _ := json.Marshal(as3Config)
	return as3Declaration(data)
}

// Function to prepare empty AS3 declaration for BIGIP Partition managed by CIS
func (am *AS3Manager) getEmptyAs3DeclarationForCISManagedPartition(partition string) as3Declaration {
	var as3Config map[string]interface{}
	baseAS3ConfigEmpty := fmt.Sprintf(baseAS3Config, am.as3Version, am.as3Release, am.as3SchemaVersion)
	_ = json.Unmarshal([]byte(baseAS3ConfigEmpty), &as3Config)
	decl := as3Config["declaration"].(map[string]interface{})

	controlObj := make(as3Control)
	controlObj.initDefault(am.userAgent)
	decl["controls"] = controlObj
	if partition != "" {
		tenantObj := make(as3Tenant)
		tenantObj.initDefault(am.defaultRouteDomain)
		decl[partition] = tenantObj
	}
	data, _ := json.Marshal(as3Config)
	return as3Declaration(data)
}

// Function to prepare tenantobjects
func (am *AS3Manager) getTenantObjects(partitions []string) string {
	var as3Config map[string]interface{}
	baseAS3ConfigEmpty := fmt.Sprintf(baseAS3Config, am.as3Version, am.as3Release, am.as3SchemaVersion)
	_ = json.Unmarshal([]byte(baseAS3ConfigEmpty), &as3Config)
	decl := as3Config["declaration"].(map[string]interface{})
	for _, partition := range partitions {
		decl[partition] = map[string]string{"class": "Tenant"}
	}
	data, _ := json.Marshal(as3Config)
	return string(data)
}

func (am *AS3Manager) getDeletedTenants(curTenantMap map[string]interface{}) []string {
	prevTenants := getTenants(am.as3ActiveConfig.unifiedDeclaration, false)
	var deletedTenants []string

	for _, tnt := range prevTenants {
		if _, found := curTenantMap[tnt]; !found {
			deletedTenants = append(deletedTenants, tnt)
		}
	}
	return deletedTenants
}

func (am *AS3Manager) getDeletedTenantsFromTenantMap(curTenantMap map[string]interface{}) []string {
	var deletedTenants []string
	for activeTenant := range am.as3ActiveConfig.tenantMap {
		if _, found := curTenantMap[activeTenant]; !found {
			deletedTenants = append(deletedTenants, activeTenant)
		}
	}
	return deletedTenants
}

// Method to delete AS3 partition using partition endpoint
func (am *AS3Manager) DeleteAS3Tenant(partition string) (bool, string) {
	emptyAS3Declaration := am.getEmptyAs3Declaration(partition)
	return am.PostManager.postConfigRequests(string(emptyAS3Declaration), am.PostManager.getAS3APIURL([]string{partition}))
}

func (am *AS3Manager) CleanAS3Tenant(partition string) (bool, string) {
	emptyAS3Declaration := am.getEmptyAs3DeclarationForCISManagedPartition(partition)
	return am.PostManager.postConfigRequests(string(emptyAS3Declaration), am.PostManager.getAS3APIURL([]string{partition}))
}

// fetchAS3Schema ...
func (am *AS3Manager) fetchAS3Schema() {
	log.Debugf("[AS3] Validating AS3 schema with  %v", as3SchemaFileName)
	am.As3SchemaLatest = am.SchemaLocalPath + as3SchemaFileName
	return
}

// configDeployer blocks on ReqChan
// whenever gets unblocked posts active configuration to BIG-IP
func (am *AS3Manager) ConfigDeployer() {
	// For the very first post after starting controller, need not wait to post
	firstPost := true
	am.unprocessableEntityStatus = false
	postDelayTimeout := time.Duration(am.PostManager.AS3PostDelay) * time.Second
	for msgReq := range am.ReqChan {
		if !firstPost && am.PostManager.AS3PostDelay != 0 {
			// Time (in seconds) that CIS waits to post the AS3 declaration to BIG-IP.
			log.Debugf("[AS3] Delaying post to BIG-IP for %v seconds", am.PostManager.AS3PostDelay)
			_ = <-time.After(postDelayTimeout)
		}

		// After postDelay expires pick up latest declaration, if available
		select {
		case msgReq = <-am.ReqChan:
		case <-time.After(1 * time.Microsecond):
		}
		posted, event := am.postAS3Declaration(msgReq.ResourceRequest)
		am.updateNetworkingConfig()

		// To handle general errors
		for !posted {
			am.unprocessableEntityStatus = true
			timeout := getTimeDurationForErrorResponse(event)
			if timeout < postDelayTimeout {
				timeout = postDelayTimeout
			}
			log.Debugf("[AS3] Error handling for event %v", event)
			posted, event = am.postOnEventOrTimeout(timeout)
			am.updateNetworkingConfig()
		}
		firstPost = false
		if event == responseStatusOk {
			am.unprocessableEntityStatus = false
		}
	}
}

func (am *AS3Manager) failureHandler() (bool, string) {
	if am.FilterTenants {
		responseStatusList := getResponseStatusList()
		for tenantName, unifiedDeclPerTenant := range am.failedContext.failedTenants {
			_, responseCode := am.PostManager.postConfigRequests(string(unifiedDeclPerTenant), am.PostManager.getAS3APIURL([]string{tenantName}))
			responseStatusList[responseCode] = responseStatusList[responseCode] + 1
			if responseCode == responseStatusOk {
				delete(am.failedContext.failedTenants, tenantName)
			}
		}
		return processResponseCodeList(responseStatusList)
	}
	return am.PostManager.postConfigRequests(string(am.as3ActiveConfig.unifiedDeclaration), am.PostManager.getAS3APIURL(nil))
}

// Helper method used by configDeployer to handle error responses received from BIG-IP
func (am *AS3Manager) postOnEventOrTimeout(timeout time.Duration) (bool, string) {
	select {
	case msgReq := <-am.ReqChan:
		return am.postAS3Declaration(msgReq.ResourceRequest)
	case <-time.After(timeout):
		return am.failureHandler()
	}
}

// Post ARP entries over response channel
func (am *AS3Manager) SendAgentResponse() {
	agRsp := am.ResourceResponse
	agRsp.IsResponseSuccessful = true
	am.postAgentResponse(MessageResponse{ResourceResponse: agRsp})
}

// Method implements posting MessageResponse on Agent Response Channel
func (am *AS3Manager) postAgentResponse(msgRsp MessageResponse) {
	select {
	case am.RspChan <- msgRsp:
	case <-am.RspChan:
		am.RspChan <- msgRsp
	}
}

// Method to verify if App Services are installed or CIS as3 version is
// compatible with BIG-IP, it will return with error if any one of the
// requirements are not met
func (am *AS3Manager) IsBigIPAppServicesAvailable() error {
	version, build, schemaVersion, err := am.PostManager.GetBigipAS3Version()
	am.as3Version = version
	as3Build := build
	am.as3SchemaVersion = schemaVersion
	am.as3Release = am.as3Version + "-" + as3Build
	if err != nil {
		log.Errorf("[AS3] %v ", err)
		return err
	}
	versionstr := version[:strings.LastIndex(version, ".")]
	bigIPAS3Version, err := strconv.ParseFloat(versionstr, 64)
	if err != nil {
		log.Errorf("[AS3] Error while converting AS3 version to float")
		return err
	}
	am.bigIPAS3Version = bigIPAS3Version
	if bigIPAS3Version >= as3SupportedVersion && bigIPAS3Version <= as3Version {
		log.Debugf("[AS3] BIGIP is serving with AS3 version: %v", version)
		return nil
	}

	if bigIPAS3Version > as3Version {
		am.as3Version = defaultAS3Version
		am.as3SchemaVersion = fmt.Sprintf("%.2f.0", as3Version)
		as3Build := defaultAS3Build
		am.as3Release = am.as3Version + "-" + as3Build
		log.Debugf("[AS3] BIGIP is serving with AS3 version: %v", bigIPAS3Version)
		return nil
	}

	return fmt.Errorf("CIS versions >= 2.0 are compatible with AS3 versions >= %v. "+
		"Upgrade AS3 version in BIGIP from %v to %v or above.", as3SupportedVersion,
		bigIPAS3Version, as3SupportedVersion)
}

func (am *AS3Manager) updateNetworkingConfig() {
	log.Debugf("[AS3] Preparing response message to response handler for arp and fdb config")
	am.SendARPEntries()
	am.SendAgentResponse()
	log.Debugf("[AS3] Sent response message to response handler for arp and fdb config")
}
