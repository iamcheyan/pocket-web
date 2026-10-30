# Test suite

Backend tests are colocated with the service in `backend/*_test.go`, where they can exercise SQLite migrations, adapters, and HTTP handlers directly. Use temporary database directories and `httptest` servers; tests must not touch the user's browser profile or external services. Root `make test` runs backend Go tests and frontend tests/type checking.

Coverage should expand with implemented behavior to include migration and restart persistence; bookmark folders and import/export; reading-list independence; history aggregation/search/deletion/private filtering; tab order/duplicate/restore/session recovery; OAuth PKCE/state and credential encryption; Download Center offline behavior; and engine lifecycle/navigation. Frontend coverage includes i18n key parity, TypeScript checking, and Vite production build. Tests must assert only observed behavior; mocks must not be counted as real integrations.
