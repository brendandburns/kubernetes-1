package master

import (
	"testing"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/expapi"
	"k8s.io/kubernetes/pkg/registry/thirdpartyresourcedata"
	"k8s.io/kubernetes/pkg/util"
)

type FakeAPIInterface struct {
	removed   []string
	installed []*expapi.ThirdPartyResource
	apis      []string
	t         *testing.T
}

func (f *FakeAPIInterface) RemoveAPI(path string) {
	f.removed = append(f.removed, path)
}

func (f *FakeAPIInterface) InstallThirdPartyAPI(rsrc *expapi.ThirdPartyResource) error {
	f.installed = append(f.installed, rsrc)
	_, group, _ := thirdpartyresourcedata.ExtractApiGroupAndKind(rsrc)
	f.apis = append(f.apis, makeThirdPartyPath(group))
	return nil
}

func (f *FakeAPIInterface) HasAPI(rsrc *expapi.ThirdPartyResource) (bool, error) {
	if f.apis == nil {
		return false, nil
	}
	_, group, _ := thirdpartyresourcedata.ExtractApiGroupAndKind(rsrc)
	path := makeThirdPartyPath(group)
	for _, api := range f.apis {
		if api ==  path {
			return true, nil
		}
	}
	return false, nil
}

func (f *FakeAPIInterface) ListThirdPartyAPIs() []string {
	return f.apis
}

func TestSyncAPIs(t *testing.T) {
	tests := []struct {
		list              *expapi.ThirdPartyResourceList
		apis              []string
		expectedInstalled []string
		expectedRemoved   []string
		name              string
	}{
		{
			list: &expapi.ThirdPartyResourceList{
				Items: []expapi.ThirdPartyResource{
					expapi.ThirdPartyResource{
						ObjectMeta: api.ObjectMeta{
							Name: "foo.example.com",
						},
					},
				},
			},
			expectedInstalled: []string{"foo.example.com"},
			name:         "simple add",
		},
		{
			list: &expapi.ThirdPartyResourceList{
				Items: []expapi.ThirdPartyResource{
					expapi.ThirdPartyResource{
						ObjectMeta: api.ObjectMeta{
							Name: "foo.example.com",
						},
					},
				},
			},
			apis: []string{
				"/thirdparty/example.com",
				"/thirdparty/example.com/v1",
			},
			name: "does nothing",
		},
		{
			list: &expapi.ThirdPartyResourceList{
				Items: []expapi.ThirdPartyResource{
					expapi.ThirdPartyResource{
						ObjectMeta: api.ObjectMeta{
							Name: "foo.example.com",
						},
					},
					expapi.ThirdPartyResource{
						ObjectMeta: api.ObjectMeta{
							Name: "foo.company.com",
						},
					},
				},
			},
			apis: []string{
				"/thirdparty/company.com",
				"/thirdparty/company.com/v1",
			},
			expectedInstalled: []string{"foo.example.com"},
			name:         "adds with existing",
		},
		{
			list: &expapi.ThirdPartyResourceList{
				Items: []expapi.ThirdPartyResource{
					expapi.ThirdPartyResource{
						ObjectMeta: api.ObjectMeta{
							Name: "foo.example.com",
						},
					},
				},
			},
			apis: []string{
				"/thirdparty/company.com",
				"/thirdparty/company.com/v1",
			},
			expectedInstalled: []string{"foo.example.com"},
			expectedRemoved:   []string{"/thirdparty/company.com", "/thirdparty/company.com/v1"},
			name:              "removes with existing",
		},
	}

	for _, test := range tests {
		fake := FakeAPIInterface{
			apis: test.apis,
			t:    t,
		}
		cntrl := ThirdPartyController{master: &fake}

		if err := cntrl.syncResourceList(test.list); err != nil {
			t.Errorf("[%s] unexpected error: %v", test.name)
		}
		if len(test.expectedInstalled) != len(fake.installed) {
			t.Errorf("[%s] unexpected installed APIs: %d, expected %d (%#v)", test.name, len(fake.installed), len(test.expectedInstalled), fake.installed[0])
			continue
		} else {
			names := util.StringSet{}
			for ix := range fake.installed {
				names.Insert(fake.installed[ix].Name)				
			}
			for _, name := range test.expectedInstalled {
				if !names.Has(name) {
					t.Errorf("[%s] missing installed API: %s", test.name, name)
				}
			}
		}
		if len(test.expectedRemoved) != len(fake.removed) {
			t.Errorf("[%s] unexpected installed APIs: %d, expected %d", test.name, len(fake.removed), len(test.expectedRemoved))
			continue
		} else {
			names := util.StringSet{}
			names.Insert(fake.removed...)
			for _, name := range test.expectedRemoved {
				if !names.Has(name) {
					t.Errorf("[%s] missing removed API: %s (%s)", test.name, name, names)
				}
			}
		}
	}
}
