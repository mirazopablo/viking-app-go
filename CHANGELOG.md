<a name="unreleased"></a>
## [Unreleased]


<a name="v2.2.0"></a>
## [v2.2.0](https://github.com/mirazopablo/viking-app-go/compare/v2.1.0...v2.2.0) (2026-09-05)

### Features
- **booking:** add dedicated endpoint for tomorrow bookings


<a name="v2.1.0"></a>
## [v2.1.0](https://github.com/mirazopablo/viking-app-go/compare/v2.0.0...v2.1.0) (2026-09-04)

### Features
- **booking:** add dedicated endpoint for today bookings


<a name="v2.0.0"></a>
## [v2.0.0](https://github.com/mirazopablo/viking-app-go/compare/v1.0.0...v2.0.0) (2026-09-04)

### Code Refactoring
- **booking:** decouple google calendar from service layer and implement dynamic slots


<a name="v1.0.0"></a>
## v1.0.0 (2026-09-04)

### Bug Fixes
- **routes:** temporarily disable automation endpoints to fix production build
- **services:** resolve pointer dereference type mismatch for work order client
- **user:** enforce transactional hard delete for user and role associations

### Code Refactoring
- **booking:** load Google credentials from base64 env var with file fallback
- **database:** remove soft delete timestamps and enforce SET NULL foreign keys
- **rbac:** standardize role entity attribute naming to english

### Features
- **api:** expose normalized GET search endpoints and query selectors
- **api:** expose budget endpoints and register router dependencies
- **api:** add REST controllers and configure routing endpoints
- **api:** Added PATCH /api/work-order/regenerate-code/:orderId protected
- **auth:** enrich login payload and JWT claims with roleName and isStaff
- **auth:** make JWT expiration configurable via environment variable
- **booking:** implement BookingRepository with duplicate slot detection
- **booking:** add Booking GORM model and request/response DTOs
- **booking:** implement BookingService with Google Calendar as source of truth
- **booking:** add BookingController with availability and create endpoints
- **bookings:** implement availability exceptions and blocking system
- **budget:** implement public unauthenticated endpoint for client budget views
- **budget:** implement hard delete for dynamic budgets
- **config:** add database and application configuration modules
- **core:** integrate notifications into work orders and routes
- **database:** implement secure interactive cli seeder tool
- **database:** implement normalized incremental search queries across repositories
- **database:** implement repositories for data persistence
- **device:** support partial PATCH updates for device owner reassignment
- **models:** add client snapshot fields and update search DTOs
- **models:** implement domain entities and database schemas
- **models:** define budget entity schema and diagnostic point entry types
- **notifications:** implement push notification service and endpoints
- **observability:** implement conditional diagnostic logging and clean production startup
- **repository:** implement GORM database operations for budgets
- **routing:** wire booking layer into router and register AutoMigrate
- **service:** implement budget calculation logic and timeline audit logging
- **services:** implement physical disk deletion and resilient DTO mappings
- **services:** implement business logic, JWT authentication and middlewares
- **services:** propagate normalized search parameters down to repository layer
- **user:** make secondary phone number optional and nullable
- **work-order:** implement cryptographic security code and recursive folder deletion
- **work-order:** implement public client status inquiry by UUID and DNI
- **work-order:** implement public client status inquiry by UUID and DNI
- **workorder:** implement historical snapshotting and resilient search

### Performance Improvements
- **api:** add lightweight user autocomplete projection and decouple diagnostic point routing
- **work-order:** implement specialized DTO projections for work order endpoints

