/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	cisv1 "github.com/F5Networks/k8s-bigip-ctlr/config/apis/cis/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeTLSProfiles implements TLSProfileInterface
type FakeTLSProfiles struct {
	Fake *FakeK8sV1
	ns   string
}

var tlsprofilesResource = schema.GroupVersionResource{Group: "k8s.nginx.org", Version: "v1", Resource: "tlsprofiles"}

var tlsprofilesKind = schema.GroupVersionKind{Group: "k8s.nginx.org", Version: "v1", Kind: "TLSProfile"}

// Get takes name of the tLSProfile, and returns the corresponding tLSProfile object, and an error if there is any.
func (c *FakeTLSProfiles) Get(name string, options v1.GetOptions) (result *cisv1.TLSProfile, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(tlsprofilesResource, c.ns, name), &cisv1.TLSProfile{})

	if obj == nil {
		return nil, err
	}
	return obj.(*cisv1.TLSProfile), err
}

// List takes label and field selectors, and returns the list of TLSProfiles that match those selectors.
func (c *FakeTLSProfiles) List(opts v1.ListOptions) (result *cisv1.TLSProfileList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(tlsprofilesResource, tlsprofilesKind, c.ns, opts), &cisv1.TLSProfileList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &cisv1.TLSProfileList{ListMeta: obj.(*cisv1.TLSProfileList).ListMeta}
	for _, item := range obj.(*cisv1.TLSProfileList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested tLSProfiles.
func (c *FakeTLSProfiles) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(tlsprofilesResource, c.ns, opts))

}

// Create takes the representation of a tLSProfile and creates it.  Returns the server's representation of the tLSProfile, and an error, if there is any.
func (c *FakeTLSProfiles) Create(tLSProfile *cisv1.TLSProfile) (result *cisv1.TLSProfile, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(tlsprofilesResource, c.ns, tLSProfile), &cisv1.TLSProfile{})

	if obj == nil {
		return nil, err
	}
	return obj.(*cisv1.TLSProfile), err
}

// Update takes the representation of a tLSProfile and updates it. Returns the server's representation of the tLSProfile, and an error, if there is any.
func (c *FakeTLSProfiles) Update(tLSProfile *cisv1.TLSProfile) (result *cisv1.TLSProfile, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(tlsprofilesResource, c.ns, tLSProfile), &cisv1.TLSProfile{})

	if obj == nil {
		return nil, err
	}
	return obj.(*cisv1.TLSProfile), err
}

// Delete takes name of the tLSProfile and deletes it. Returns an error if one occurs.
func (c *FakeTLSProfiles) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(tlsprofilesResource, c.ns, name), &cisv1.TLSProfile{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeTLSProfiles) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(tlsprofilesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &cisv1.TLSProfileList{})
	return err
}

// Patch applies the patch and returns the patched tLSProfile.
func (c *FakeTLSProfiles) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *cisv1.TLSProfile, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(tlsprofilesResource, c.ns, name, pt, data, subresources...), &cisv1.TLSProfile{})

	if obj == nil {
		return nil, err
	}
	return obj.(*cisv1.TLSProfile), err
}