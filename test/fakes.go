// Package test provides a minimal in-process fake of the nulink API for
// local development against the fco-apiserver. It serves the VM endpoints
// with canned data; mutation endpoints return a generic request response
// and /validate endpoints always succeed.
package test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/fi-ts/nulink-go/client"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// Start boots the fake nulink API and returns the generated client dialed
// at it, the server URL and a cleanup func that stops the server.
func Start() (*client.Client, string, func(), error) {
	mux := http.NewServeMux()
	registerVMHandlers(mux)

	// Enable h2c so gRPC-over-HTTP clients work against the fake.
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

func registerVMHandlers(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/vm/location", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, defaultLocations())
	})

	mux.HandleFunc("GET /api/v1/vm/os", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, defaultOSs())
	})

	mux.HandleFunc("GET /api/v1/vm/stagetype", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, defaultStageTypes())
	})

	mux.HandleFunc("GET /api/v1/vm/vlan", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, defaultVlans())
	})

	// Optional instance_uuid query narrows the result to a single VM;
	// unknown UUIDs get a 404.
	mux.HandleFunc("GET /api/v1/vm/instance", func(w http.ResponseWriter, r *http.Request) {
		instanceUUID := r.URL.Query().Get("instance_uuid")
		if instanceUUID == "" {
			writeJSON(w, defaultVMs())
			return
		}
		for i := range defaultVMs().Instances {
			// Try Windows first: the union helpers unmarshal any record into
			// either struct; domain_uuid marks windows records.
			if w_, err := defaultVMs().Instances[i].AsNucleusDBWindowsVMs(); err == nil && w_.DomainUuid != nil && w_.VmUuid != nil && w_.VmUuid.String() == instanceUUID {
				writeJSON(w, vmList(w_))
				return
			}
			if l, err := defaultVMs().Instances[i].AsNucleusDBLinuxVMs(); err == nil && l.VmUuid != nil && l.VmUuid.String() == instanceUUID {
				writeJSON(w, vmList(l))
				return
			}
		}
		http.Error(w, "vm instance not found", http.StatusNotFound)
	})

	mux.HandleFunc("POST /api/v1/vm/instance/linux", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, genericResponse("create_vm_application_server", linuxVMUUID))
	})
	mux.HandleFunc("POST /api/v1/vm/instance/linux/create/validate", validateOK)

	mux.HandleFunc("POST /api/v1/vm/instance/windows", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, genericResponse("create_vm_application_server", windowsVMUUID))
	})
	mux.HandleFunc("POST /api/v1/vm/instance/windows/create/validate", validateOK)

	mux.HandleFunc("DELETE /api/v1/vm/instance/{instance_uuid}", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, genericResponse("delete_vm", r.PathValue("instance_uuid")))
	})

	mux.HandleFunc("POST /api/v1/vm/instance/linux/{instance_uuid}/disk", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, genericResponse("update_vm_create_disk", r.PathValue("instance_uuid")))
	})
	mux.HandleFunc("POST /api/v1/vm/instance/linux/{instance_uuid}/disk/create/validate", validateOK)

	mux.HandleFunc("POST /api/v1/vm/instance/windows/{instance_uuid}/disk", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, genericResponse("update_vm_create_disk", r.PathValue("instance_uuid")))
	})
	mux.HandleFunc("POST /api/v1/vm/instance/windows/{instance_uuid}/disk/create/validate", validateOK)

	mux.HandleFunc("PUT /api/v1/vm/instance/{instance_uuid}/disk/{disk_uuid}", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, genericResponse("update_vm_update_disk", r.PathValue("disk_uuid")))
	})
	mux.HandleFunc("POST /api/v1/vm/instance/{instance_uuid}/disk/{disk_uuid}/update/validate", validateOK)

	mux.HandleFunc("DELETE /api/v1/vm/instance/{instance_uuid}/disk/{disk_uuid}", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, genericResponse("update_vm_delete_disk", r.PathValue("disk_uuid")))
	})
	mux.HandleFunc("POST /api/v1/vm/instance/{instance_uuid}/disk/{disk_uuid}/delete/validate", validateOK)

	mux.HandleFunc("POST /api/v1/vm/instance/{instance_uuid}/ip", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, genericResponse("update_vm_create_ip", r.PathValue("instance_uuid")))
	})
	mux.HandleFunc("POST /api/v1/vm/instance/{instance_uuid}/ip/create/validate", validateOK)

	mux.HandleFunc("PUT /api/v1/vm/instance/{instance_uuid}/ip/{interface_uuid}", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, genericResponse("update_vm_update_ip", r.PathValue("interface_uuid")))
	})
	mux.HandleFunc("POST /api/v1/vm/instance/{instance_uuid}/ip/{interface_uuid}/update/validate", validateOK)

	mux.HandleFunc("DELETE /api/v1/vm/instance/{instance_uuid}/ip/{interface_uuid}", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, genericResponse("update_vm_delete_ip", r.PathValue("interface_uuid")))
	})

	mux.HandleFunc("PUT /api/v1/vm/instance/{instance_uuid}/performanceclass", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, genericResponse("update_vm_performanceclass", r.PathValue("instance_uuid")))
	})
	mux.HandleFunc("PUT /api/v1/vm/instance/{instance_uuid}/serviceclass", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, genericResponse("update_vm_serviceclass", r.PathValue("instance_uuid")))
	})
	mux.HandleFunc("PUT /api/v1/vm/instance/{instance_uuid}/contact", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, genericResponse("update_vm_update_contact", r.PathValue("instance_uuid")))
	})
}

var validateOK = func(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, &client.ValidationSuccessResponseModel{Success: true})
}

// genericResponse builds a canned response for a mutation endpoint;
// requestType must be a valid upstream RequestTypeConfig value.
func genericResponse(requestType, itemUUID string) *client.GenericRequestResponseModel {
	return &client.GenericRequestResponseModel{
		AutomationId:   1,
		AutomationType: "vm_instance",
		ItemUuid:       uuid.MustParse(itemUUID),
		RequestStatus:  client.RequestStatusConfig("Requested"),
		RequestType:    client.RequestTypeConfig(requestType),
	}
}

func vmList(vm any) *client.VMInstanceGetVMsResponseModel {
	out := &client.VMInstanceGetVMsResponseModel{}
	switch v := vm.(type) {
	case client.NucleusDBLinuxVMs:
		item := client.VMInstanceGetVMsResponseModel_Instances_Item{}
		if err := item.FromNucleusDBLinuxVMs(v); err != nil {
			panic(err)
		}
		out.Instances = []client.VMInstanceGetVMsResponseModel_Instances_Item{item}
	case client.NucleusDBWindowsVMs:
		item := client.VMInstanceGetVMsResponseModel_Instances_Item{}
		if err := item.FromNucleusDBWindowsVMs(v); err != nil {
			panic(err)
		}
		out.Instances = []client.VMInstanceGetVMsResponseModel_Instances_Item{item}
	default:
		panic(fmt.Sprintf("unsupported vm type %T", vm))
	}
	return out
}

func defaultLocations() *client.VMInstanceGetLocationsResponeModel {
	return &client.VMInstanceGetLocationsResponeModel{
		Locations: []client.NucleusDBLocations{
			{LocationUuid: uuidPtr("8d0f3da7-2e4b-6275-9b3a-910b733acf71"), LocationTitle: "Nürnberg", Tenant: "ftap"},
			{LocationUuid: uuidPtr("c5f0b5d0-77b8-10b1-63dd-7f6a624e00b5"), LocationTitle: "Stuttgart", Tenant: "ftap"},
		},
	}
}

func defaultOSs() *client.VMInstanceGetOSSResponseModel {
	return &client.VMInstanceGetOSSResponseModel{
		Oss: []client.NucleusDBOss{
			{OsTitle: "Linux Red Hat Enterprise 7", OsType: "linux", OsVersion: "7_RHEL", OsUuid: uuidPtr("0c1b9938-258a-1308-4e4d-988d0742de4f")},
			{OsTitle: "Windows Server 2016 Standard", OsType: "windows", OsVersion: "2016_STD", OsUuid: uuidPtr("0475f7bd-f538-2b71-07f3-3439ad4b5000")},
			{OsTitle: "Linux Red Hat Enterprise 8", OsType: "linux", OsVersion: "8_RHEL", OsUuid: uuidPtr("e3ebdbbe-a307-e99e-9c6b-4f35e1873c80")},
			{OsTitle: "Windows Server 2019 Standard", OsType: "windows", OsVersion: "2019_STD", OsUuid: uuidPtr("10ab5f22-80af-5bee-8787-db6d8c6a0dd3")},
			{OsTitle: "Windows Server 2022 Standard", OsType: "windows", OsVersion: "2022_STD", OsUuid: uuidPtr("61121fe8-d0f8-5098-66f4-56be969626d8")},
			{OsTitle: "Linux Red Hat Enterprise 9", OsType: "linux", OsVersion: "9_RHEL", OsUuid: uuidPtr("29512413-f6c3-4837-ecfc-1f8072064ac7")},
			{OsTitle: "Windows 11 Enterprise", OsType: "vdi", OsVersion: "11_ENT", OsUuid: uuidPtr("89038441-837e-2a79-1181-ed0ac1fc5e00")},
			{OsTitle: "Windows Server 2025 Standard", OsType: "windows", OsVersion: "2025_STD", OsUuid: uuidPtr("6d66471b-a89f-25ef-5bf2-1eb4f4155676")},
			{OsTitle: "Kali 2025.2 Appliance", OsType: "linux", OsVersion: "Kali_APPL", OsUuid: uuidPtr("3108baf3-bb86-8258-5afd-6146d8870e10")},
			{OsTitle: "Kiteworks Appliance", OsType: "linux-appliance", OsVersion: "Kite_APPL", OsUuid: uuidPtr("f6718c16-9319-f494-35be-46f5a4538ab1")},
		},
	}
}

func defaultStageTypes() *client.VMInstanceGetStageTypesResponseModel {
	return &client.VMInstanceGetStageTypesResponseModel{
		StageTypes: []client.NucleusDBStageTypes{
			{StageTypeUuid: uuidPtr("f04ac2b8-8157-352a-c0f1-36a59a58d841"), StageTypeTitle: "Entwicklung"},
			{StageTypeUuid: uuidPtr("410cda2a-5810-135c-f25c-48a8200ab112"), StageTypeTitle: "Produktion"},
			{StageTypeUuid: uuidPtr("c3739ef4-11f1-eaa8-4fcc-717aeaad508e"), StageTypeTitle: "Test"},
			{StageTypeUuid: uuidPtr("853b452e-bb72-a743-f1b5-b308387cdb41"), StageTypeTitle: "Abnahme"},
		},
	}
}

func defaultVlans() *client.VMInstanceGetVlansResponseModel {
	return &client.VMInstanceGetVlansResponseModel{
		Vlans: []client.NucleusDBVlans{
			{
				VlanUuid:       uuidPtr("ee8e4fe8-ecda-45bd-467f-c84a9a17704e"),
				VlanId:         2001,
				PodTitle:       "POD01",
				PodUuid:        uuidPtr("ef2465a1-0eb9-4bc4-0645-da84f1397482"),
				SubnetTitle:    "pg-mgm-pod01-vm-vlan2001",
				Subnet:         "100.64.1.0/24",
				Subnetmask:     "255.255.255.0",
				Gateway:        "100.64.1.1",
				IpRangeStart:   "100.64.1.33",
				IpRangeEnd:     "100.64.1.254",
				UseDhcp:        false,
				StageTypeTitle: "Produktion",
				StageTypeUuid:  uuidPtr("410cda2a-5810-135c-f25c-48a8200ab112"),
				Tenant:         "fac7f4b6-bd59-fef7-cd45-dcaddb43e523",
			},
			{
				VlanUuid:       uuidPtr("d35e8a17-6292-aacd-7905-d228ef560f68"),
				VlanId:         2008,
				PodTitle:       "POD01",
				PodUuid:        uuidPtr("ef2465a1-0eb9-4bc4-0645-da84f1397482"),
				SubnetTitle:    "pg-svc-pod01-vm-vlan2008",
				Subnet:         "100.64.8.0/24",
				Subnetmask:     "255.255.255.0",
				Gateway:        "100.64.8.1",
				IpRangeStart:   "100.64.8.33",
				IpRangeEnd:     "100.64.8.254",
				UseDhcp:        false,
				StageTypeTitle: "Produktion",
				StageTypeUuid:  uuidPtr("410cda2a-5810-135c-f25c-48a8200ab112"),
				Tenant:         "fac7f4b6-bd59-fef7-cd45-dcaddb43e523",
			},
		},
	}
}

const (
	windowsVMUUID = "9037fff6-0ef0-339a-443a-4f37a53cdf59"
	linuxVMUUID   = "6d9306ed-7bb2-c8b3-2f9c-ac1a70cf489c"
)

func defaultVMs() *client.VMInstanceGetVMsResponseModel {
	windows := client.NucleusDBWindowsVMs{
		VmUuid:        uuidPtr(windowsVMUUID),
		VmFqdn:        "vm-win01.example.com",
		Status:        "Active",
		StageTypeUuid: uuid.MustParse("410cda2a-5810-135c-f25c-48a8200ab112"),
		Interfaces:    &[]client.NucleusDBInterface{{InterfaceUuid: uuidPtr("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"), Ipaddress: "10.0.0.11", Macaddress: "AA:BB:CC:DD:EE:01"}},
		Encrypted:     false,
		Disks:         &[]client.NucleusDBWindowsDisk{{AutoExtend: true, DiskSizeGb: 100, DiskUuid: uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"), Driveletter: "C"}},
		DomainUuid:    uuidPtr("53ab00bc-c863-78da-869f-f80063372c74"),
		DomainFqdn:    strPtr("corp.example.com"),
		ProjectUuid:   uuid.MustParse("e8b24441-2712-4dc0-878f-503f53102b28"),
		Tenant:        "ftap",
		Contract:      true,
		Serviceclass:  "SZ3",
		Availability:  "V3",
		Cpu:           8,
		Ram:           32,
		Storageclass:  "mgmtfabric",
		ServiceId:     "application_server",
		ServiceTitle:  "Application Server",
		ContactUuid:   uuid.MustParse("43920c7c-6f51-f3cb-ec54-4c226556118f"),
		OsUuid:        uuid.MustParse("61121fe8-d0f8-5098-66f4-56be969626d8"),
		OsTitle:       "Windows Server 2022 Standard",
		OsType:        "windows",
		OrderNumber:   "12345",
		Labels:        mapPtr(map[string]interface{}{"accounting_key": "AK-WIN-1"}),
	}
	linux := client.NucleusDBLinuxVMs{
		VmUuid:        uuidPtr(linuxVMUUID),
		VmFqdn:        "vm-lin01.example.com",
		Status:        "Active",
		StageTypeUuid: uuid.MustParse("410cda2a-5810-135c-f25c-48a8200ab112"),
		Interfaces:    &[]client.NucleusDBInterface{{InterfaceUuid: uuidPtr("cccccccc-cccc-cccc-cccc-cccccccccccc"), Ipaddress: "10.0.0.21", Macaddress: "AA:BB:CC:DD:EE:02"}},
		Encrypted:     false,
		Disks:         &[]client.NucleusDBLinuxDisk{{AutoExtend: true, DiskSizeGb: 50, DiskUuid: uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd"), Mountpoint: "/data"}},
		LdapUuid:      uuidPtr("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"),
		LdapFqdn:      strPtr("ldap.example.com"),
		ProjectUuid:   uuid.MustParse("e8b24441-2712-4dc0-878f-503f53102b28"),
		Tenant:        "ftap",
		Contract:      false,
		Serviceclass:  "SZ1",
		Availability:  "V1",
		Cpu:           2,
		Ram:           4,
		Storageclass:  "mgmtfabric",
		ServiceId:     "application_server",
		ServiceTitle:  "Application Server",
		ContactUuid:   openapi_types.UUID{},
		OsUuid:        uuid.MustParse("29512413-f6c3-4837-ecfc-1f8072064ac7"),
		OsTitle:       "Linux Red Hat Enterprise 9",
		OsType:        "linux",
		OrderNumber:   "12345",
		Labels:        mapPtr(map[string]interface{}{"accounting_key": "AK-LIN-1"}),
	}

	winItem := client.VMInstanceGetVMsResponseModel_Instances_Item{}
	if err := winItem.FromNucleusDBWindowsVMs(windows); err != nil {
		panic(err)
	}
	linItem := client.VMInstanceGetVMsResponseModel_Instances_Item{}
	if err := linItem.FromNucleusDBLinuxVMs(linux); err != nil {
		panic(err)
	}
	return &client.VMInstanceGetVMsResponseModel{
		Instances: []client.VMInstanceGetVMsResponseModel_Instances_Item{winItem, linItem},
	}
}

func strPtr(s string) *string {
	return &s
}

func uuidPtr(s string) *openapi_types.UUID {
	u, _ := uuid.Parse(s)
	return &u
}

func mapPtr(m map[string]interface{}) *map[string]interface{} {
	return &m
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
