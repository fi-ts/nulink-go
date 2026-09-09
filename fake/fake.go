// Package fake provides an in-process fake of the nulink API for tests and
// local development.
// the full stack (generated client -> HTTP -> handler) runs in-process against canned
// data, nothing below the handler level is stubbed.
//
// Consumers can dial the generated client at the fake's URL and exercise real
// transport, JSON encoding and response parsing end-to-end.
package fake

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/fi-ts/nulink-go/client"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// Start boots an in-process fake of the nulink API and returns the raw
// generated client dialed at it, the server URL and a cleanup func that
// stops the server.
func Start() (*client.Client, string, func(), error) {
	mux := http.NewServeMux()

	registerVMHandlers(mux)

	// Enable h2c so connect clients using the gRPC protocol work against the fake.
	protocols := &http.Protocols{}
	protocols.SetHTTP1(true)
	protocols.SetHTTP2(true)
	protocols.SetUnencryptedHTTP2(true)

	server := httptest.NewUnstartedServer(mux)
	server.Config.Protocols = protocols
	server.Start()

	c, err := client.NewClient(server.URL)
	if err != nil {
		server.Close()
		return nil, "", func() {}, fmt.Errorf("create nulink client: %w", err)
	}

	return c, server.URL, server.Close, nil
}

// registerVMHandlers serves the VM-related endpoints from
// spec/nulink-openapi.json with canned data. Add more handlers here as
// consumers need them.
func registerVMHandlers(mux *http.ServeMux) {
	// POST /api/v1/vm/location/list - Listet alle verfügbaren VM Locations.
	mux.HandleFunc("POST /api/v1/vm/location/list", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, &client.VMInstanceGetLocationsResponeModel{
			Locations: []client.NucleusDBLocations{
				{LocationUuid: uuidPtr("8d0f3da7-2e4b-6275-9b3a-910b733acf71"), Title: "Nürnberg"},
				{LocationUuid: uuidPtr("c5f0b5d0-77b8-10b1-63dd-7f6a624e00b5"), Title: "Stuttgart"},
			},
		})
	})

	// POST /api/v1/vm/os/list - Listet alle verfügbaren VM OSs.
	mux.HandleFunc("POST /api/v1/vm/os/list", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, &client.VMInstanceGetOSSResponseModel{
			Oss: []client.NucleusDBOss{
				{OsTitle: "Ubuntu 24.04 LTS", OsType: "linux", OsVersion: "24.04"},
				{OsTitle: "Debian 12", OsType: "linux", OsVersion: "12"},
				{OsTitle: "Rocky Linux 9", OsType: "linux", OsVersion: "9"},
				{OsTitle: "Windows Server 2022", OsType: "windows", OsVersion: "2022"},
				{OsTitle: "Windows Server 2025", OsType: "windows", OsVersion: "2025"},
			},
		})
	})

	// POST /api/v1/vm/stagetype/list - Listet alle Stage Types.
	mux.HandleFunc("POST /api/v1/vm/stagetype/list", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, &client.VMInstanceGetStageTypesResponseModel{
			StageTypes: []client.NucleusDBStageTypes{
				{StageTypeUuid: uuidPtr("f04ac2b8-8157-352a-c0f1-36a59a58d841"), StageTypeTitle: "Entwicklung"},
				{StageTypeUuid: uuidPtr("410cda2a-5810-135c-f25c-48a8200ab112"), StageTypeTitle: "Produktion"},
				{StageTypeUuid: uuidPtr("c3739ef4-11f1-eaa8-4fcc-717aeaad508e"), StageTypeTitle: "Test"},
				{StageTypeUuid: uuidPtr("853b452e-bb72-a743-f1b5-b308387cdb41"), StageTypeTitle: "Abnahme"},
			},
		})
	})

	// POST /api/v1/vm/vlan/list - Listet alle VLANs eines Tenants.
	mux.HandleFunc("POST /api/v1/vm/vlan/list", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, &client.VMInstanceGetVlansResponseModel{
			Vlans: []client.NucleusDBVlans{
				{
					VlanUuid:       uuidPtr("77777777-7777-7777-7777-777777777777"),
					VlanId:         100,
					PodTitle:       "Pod A",
					PodUuid:        uuidPtr("88888888-8888-8888-8888-888888888888"),
					SubnetTitle:    "Subnet A",
					Subnet:         "10.0.0.0/24",
					Subnetmask:     "255.255.255.0",
					Gateway:        "10.0.0.1",
					IpRangeStart:   "10.0.0.10",
					IpRangeEnd:     "10.0.0.200",
					UseDhcp:        true,
					StageTypeTitle: "Produktion",
					StageTypeUuid:  uuidPtr("410cda2a-5810-135c-f25c-48a8200ab112"),
					TenantTitle:    "ftap",
					TenantUuid:     "ftap",
				},
			},
		})
	})
}

// uuidPtr returns a pointer to the given UUID string, for use with the
// nullable (*openapi_types.UUID) fields of the generated client models.
// The canned values are valid UUIDs, so parse errors are not expected.
func uuidPtr(s string) *openapi_types.UUID {
	u, _ := uuid.Parse(s)
	return &u
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
