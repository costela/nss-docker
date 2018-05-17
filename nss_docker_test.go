/*
Copyright © 2018 Leo Antunes <leo@costela.net>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package main

import (
	"context"
	"reflect"
	"testing"

	"docker.io/go-docker/api/types/network"

	"docker.io/go-docker/api/types"
)

type testClient struct{}

func (testClient) ContainerList(_ context.Context, _ types.ContainerListOptions) ([]types.Container, error) {
	return []types.Container{
		types.Container{
			ID:     "foo",
			Labels: map[string]string{},
			Names: []string{
				"/someservice",
			},
		},
		types.Container{
			ID: "bar",
			Labels: map[string]string{
				"com.docker.compose.project": "someproject",
			},
			Names: []string{
				"/someproject_someotherservice_1",
			},
		},
		types.Container{
			ID: "baz",
			Labels: map[string]string{
				"com.docker.compose.project": "someotherproject",
			},
			Names: []string{
				"/someotherproject_someotherservice_1",
			},
		},
	}, nil
}

func (testClient) ContainerInspect(_ context.Context, name string) (types.ContainerJSON, error) {
	switch name {
	case "foo":
		return types.ContainerJSON{
			NetworkSettings: &types.NetworkSettings{
				Networks: map[string]*network.EndpointSettings{
					"default": &network.EndpointSettings{
						IPAddress: "1.2.3.4",
						Aliases: []string{
							"somealias",
						},
					},
				},
			},
		}, nil
	case "bar":
		return types.ContainerJSON{
			NetworkSettings: &types.NetworkSettings{
				Networks: map[string]*network.EndpointSettings{
					"default": &network.EndpointSettings{
						IPAddress: "2.3.4.5",
						Aliases: []string{
							"someotheralias",
							"nonuniquealias",
						},
					},
				},
			},
		}, nil
	case "baz":
		return types.ContainerJSON{
			NetworkSettings: &types.NetworkSettings{
				Networks: map[string]*network.EndpointSettings{
					"default": &network.EndpointSettings{
						IPAddress: "3.4.5.6",
						Aliases: []string{
							"yetanotheralias",
							"nonuniquealias",
						},
					},
				},
			},
		}, nil
	default:
		panic("whaaa?")
	}
}

func Test_queryDockerForName(t *testing.T) {
	type args struct {
		fqdn string
	}
	tests := []struct {
		name          string
		config        configStruct
		args          args
		wantAliases   []string
		wantAddresses []string
		wantErr       bool
	}{
		{
			name:   "individual container without suffix",
			config: config,
			args: args{
				fqdn: "someservice",
			},
			wantAliases:   []string{"someservice.docker", "somealias.docker"},
			wantAddresses: []string{"1.2.3.4"},
			wantErr:       false,
		},
		{
			name:   "nonexistent container",
			config: config,
			args: args{
				fqdn: "somenonservice",
			},
			wantAliases:   []string{},
			wantAddresses: []string{},
			wantErr:       false,
		},
		{
			name:   "fqdn search",
			config: config,
			args: args{
				fqdn: "someservice.docker",
			},
			wantAliases:   []string{"someservice.docker", "somealias.docker"},
			wantAddresses: []string{"1.2.3.4"},
			wantErr:       false,
		},
		{
			name:   "fqdn project search",
			config: config,
			args: args{
				fqdn: "someotheralias.someproject.docker",
			},
			wantAliases:   []string{"someproject_someotherservice_1.someproject.docker", "someotheralias.someproject.docker", "nonuniquealias.someproject.docker"},
			wantAddresses: []string{"2.3.4.5"},
			wantErr:       false,
		},
		{
			name:   "non-unique alias inside project",
			config: config,
			args: args{
				fqdn: "nonuniquealias.someproject.docker",
			},
			wantAliases:   []string{"someproject_someotherservice_1.someproject.docker", "someotheralias.someproject.docker", "nonuniquealias.someproject.docker"},
			wantAddresses: []string{"2.3.4.5"},
			wantErr:       false,
		},
		{
			name:   "non-unique alias without project",
			config: configStruct{Suffix: ".docker", IncludeComposeProject: false},
			args: args{
				fqdn: "nonuniquealias.docker",
			},
			wantAliases: []string{
				"someproject_someotherservice_1.docker", "someotheralias.docker", "nonuniquealias.docker",
				"someotherproject_someotherservice_1.docker", "yetanotheralias.docker", "nonuniquealias.docker",
			},
			wantAddresses: []string{"2.3.4.5", "3.4.5.6"},
			wantErr:       false,
		},
		{
			name:   "individual container with custom suffix",
			config: configStruct{Suffix: ".mysuffix"},
			args: args{
				fqdn: "someservice.mysuffix",
			},
			wantAliases:   []string{"someservice.mysuffix", "somealias.mysuffix"},
			wantAddresses: []string{"1.2.3.4"},
			wantErr:       false,
		},
		{
			name:   "fqdn project search with custom suffix",
			config: configStruct{Suffix: ".mysuffix", IncludeComposeProject: true},
			args: args{
				fqdn: "someotheralias.someproject.mysuffix",
			},
			wantAliases:   []string{"someproject_someotherservice_1.someproject.mysuffix", "someotheralias.someproject.mysuffix", "nonuniquealias.someproject.mysuffix"},
			wantAddresses: []string{"2.3.4.5"},
			wantErr:       false,
		},
		{
			name:   "fqdn project search with custom suffix and no project",
			config: configStruct{Suffix: ".mysuffix", IncludeComposeProject: false},
			args: args{
				fqdn: "someotheralias.someproject.mysuffix",
			},
			wantAliases:   []string{},
			wantAddresses: []string{},
			wantErr:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// temporarily override config global
			defer func(cfg configStruct) {
				config = cfg
			}(config)
			config = tt.config

			gotAliases, gotAddresses, err := queryDockerForName(testClient{}, tt.args.fqdn)
			if (err != nil) != tt.wantErr {
				t.Errorf("queryDockerForName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotAliases, tt.wantAliases) {
				t.Errorf("queryDockerForName() gotAliases = %#v, want %#v", gotAliases, tt.wantAliases)
			}
			if !reflect.DeepEqual(gotAddresses, tt.wantAddresses) {
				t.Errorf("queryDockerForName() gotAddresses = %#v, want %#v", gotAddresses, tt.wantAddresses)
			}
		})
	}
}
