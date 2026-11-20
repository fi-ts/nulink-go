release:: generate-client;

.PHONY: generate-client
generate-client:
	go generate ./...

# --- Fetching spec and validations from a running nulink server ---
# Required env: NULINK_SERVER_URL (with trailing slash), NULINK_USER, NULINK_PW

# $(call nulink-fetch,<api path>,<output file>)
# Logs in, fetches the path and only replaces the output file on success.
define nulink-fetch
	@test -n "$(NULINK_SERVER_URL)" || { echo "NULINK_SERVER_URL is not set"; exit 1; }
	@test -n "$(NULINK_USER)" || { echo "NULINK_USER is not set"; exit 1; }
	@test -n "$$NULINK_PW" || { echo "NULINK_PW is not set"; exit 1; }
	@set -e; \
	token=$$(curl -fsS -X POST "$(NULINK_SERVER_URL)api/v1/login/access-token" \
		-H 'accept: application/json' \
		--data-urlencode "grant_type=password" \
		--data-urlencode "username=$(NULINK_USER)" \
		--data-urlencode "password=$$NULINK_PW" | jq --raw-output .access_token); \
	test -n "$$token" -a "$$token" != "null" || { echo "login failed"; exit 1; }; \
	curl -fsS "$(NULINK_SERVER_URL)$(1)" \
		-H 'accept: application/json' \
		-H "Authorization: Bearer $$token" \
		-H 'Content-Type: application/json' > $(2).tmp || { rm -f $(2).tmp; exit 1; }; \
	mv $(2).tmp $(2); \
	echo "wrote $(2)"
endef

.PHONY: fetch-spec
fetch-spec:
	$(call nulink-fetch,api/v1/openapi.json,spec/openapi.json)

.PHONY: fetch-validations
fetch-validations:
	$(call nulink-fetch,api/v1/mvm/validations,spec/validations.json)

# Fetch both files, regenerate the client and verify the versions agree.
.PHONY: update-spec
update-spec: fetch-spec fetch-validations generate-client
	go test ./spec/...
