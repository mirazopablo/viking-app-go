package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mirazopablo/viking-app-go/controllers"
	"github.com/mirazopablo/viking-app-go/middlewares"
	"github.com/mirazopablo/viking-app-go/repositories"
	"github.com/mirazopablo/viking-app-go/services"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	// Blank import required for Swagger documentation initialization
	_ "github.com/mirazopablo/viking-app-go/docs"
)

// CORSMiddleware configures CORS headers and handles preflight OPTIONS requests.
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// SetupRouter initializes Gin engine, middleware, controllers, and routes.
func SetupRouter() *gin.Engine {
	// Set Gin to ReleaseMode to suppress debug warnings and route dumping in production
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()

	// Attach custom conditional logger and panic recovery middlewares
	r.Use(middlewares.CustomLoggerMiddleware())
	r.Use(middlewares.CustomRecoveryMiddleware())

	// Enable CORS Middleware for Swagger UI and frontend applications
	r.Use(CORSMiddleware())

	// Initialize Repositories
	roleRepo := repositories.NewRoleRepository()
	userRepo := repositories.NewUserRepository()
	deviceRepo := repositories.NewDeviceRepository()
	workOrderRepo := repositories.NewWorkOrderRepository()
	diagnosticPointRepo := repositories.NewDiagnosticPointRepository()

	// Initialize Services
	roleService := services.NewRoleService(roleRepo)
	jwtService := services.NewJWTService()
	userService := services.NewUserService(userRepo, roleRepo, jwtService)
	deviceService := services.NewDeviceService(deviceRepo, userService)
	workOrderService := services.NewWorkOrderService(workOrderRepo, userRepo, deviceRepo, diagnosticPointRepo)
	diagnosticPointService := services.NewDiagnosticPointService(diagnosticPointRepo, workOrderRepo, userRepo)

	// Initialize Controllers
	homeCtrl := controllers.NewHomeController()
	roleCtrl := controllers.NewRoleController(roleService)
	fileCtrl := controllers.NewFileController()
	authCtrl := controllers.NewAuthController(userService)
	userCtrl := controllers.NewUserController(userService, jwtService)
	deviceCtrl := controllers.NewDeviceController(deviceService)
	userRoleCtrl := controllers.NewUserRoleController(roleService)
	workOrderCtrl := controllers.NewWorkOrderController(workOrderService)
	diagnosticPointCtrl := controllers.NewDiagnosticPointController(diagnosticPointService)

	// Swagger UI Route (Always Public)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// =========================================================================
	// PUBLIC ROUTES (No Bearer token required)
	// =========================================================================
	publicAuth := r.Group("/auth")
	{
		// Login is strictly public to generate tokens
		publicAuth.POST("/login", authCtrl.AuthenticateUser)
		// Validate is public so frontends can verify token validity
		publicAuth.GET("/validate", authCtrl.ValidateToken)

		// File serving is public so native Image components in apps can display evidence without custom headers
		publicAuth.GET("/uploads/*filepath", fileCtrl.ServeFile)

		// NOTE FOR DEPLOYMENT: If you need to create your initial admin user/role
		// on a clean server, temporarily move the /signup or /roles routes here!
	}

	publicWorkOrder := r.Group("/public/work-order")
	{
		publicWorkOrder.POST("/status", workOrderCtrl.GetPublicStatus)
		publicWorkOrder.POST("/status-by-dni", workOrderCtrl.GetPublicStatusByDni)
	}

	// =========================================================================
	// PRIVATE ROUTES (Protected by Bearer JWT Middleware)
	// =========================================================================
	privateApi := r.Group("/api")
	privateApi.Use(middlewares.AuthMiddleware(jwtService))
	{
		// Home Controller
		privateApi.GET("/", homeCtrl.Greeting)

		// User Controller Endpoints
		userGroup := privateApi.Group("/user")
		{
			userGroup.POST("/save", userCtrl.SaveUser)
			userGroup.PATCH("/update/:id", userCtrl.UpdateUser)
			userGroup.PUT("/update/:id", userCtrl.UpdateUser)
			userGroup.GET("/search", userCtrl.SearchUser)
			userGroup.GET("/autocomplete", userCtrl.AutocompleteUser)
			userGroup.GET("/current", userCtrl.GetCurrentUser)
			userGroup.DELETE("/delete/:id", userCtrl.DeleteUser)
		}

		// Device Controller Endpoints
		deviceGroup := privateApi.Group("/device")
		{
			deviceGroup.POST("/save", deviceCtrl.RegisterDevice)
			deviceGroup.PATCH("/update/:id", deviceCtrl.UpdateDevice)
			deviceGroup.PUT("/update/:id", deviceCtrl.UpdateDevice)
			deviceGroup.GET("/search", deviceCtrl.SearchDevice)
			deviceGroup.DELETE("/delete/:id", deviceCtrl.DeleteDevice)
		}

		// User Role Controller Endpoints
		userRoleGroup := privateApi.Group("/user-roles")
		{
			userRoleGroup.GET("/user-permission", userRoleCtrl.GetUserPermission)
			userRoleGroup.GET("/is-staff", userRoleCtrl.IsUserStaff)
		}

		// Work Order Controller Endpoints
		workOrderGroup := privateApi.Group("/work-order")
		{
			workOrderGroup.POST("/save", workOrderCtrl.SaveWorkOrder)
			workOrderGroup.PATCH("/update/:orderId", workOrderCtrl.UpdateWorkOrderStatus)
			workOrderGroup.PATCH("/regenerate-code/:orderId", workOrderCtrl.RegenerateSecurityCode)
			workOrderGroup.GET("/search", workOrderCtrl.SearchWorkOrder)
			workOrderGroup.GET("/:orderId", workOrderCtrl.GetWorkOrderByID)
			workOrderGroup.DELETE("/delete/:orderId", workOrderCtrl.DeleteWorkOrder)
		}

		// Diagnostic Point Controller Endpoints
		diagnosticPointGroup := privateApi.Group("/diagnostic-points")
		{
			diagnosticPointGroup.POST("/add", diagnosticPointCtrl.AddDiagnosticPoint)
			diagnosticPointGroup.GET("/by-work-order/:workOrderId/client/:clientId", diagnosticPointCtrl.GetDiagnosticPointsByWorkOrderAndClient)
			diagnosticPointGroup.GET("/by-work-order/:workOrderId", diagnosticPointCtrl.GetDiagnosticPointsByWorkOrder)
			diagnosticPointGroup.DELETE("/delete/:id", diagnosticPointCtrl.DeleteDiagnosticPoint)
		}
	}

	privateAuth := r.Group("/auth")
	privateAuth.Use(middlewares.AuthMiddleware(jwtService))
	{
		// Signup is protected so only logged-in admins can register new users
		privateAuth.POST("/signup", authCtrl.RegisterUser)

		// Role Controller Endpoints (Under /auth/roles as specified in openapi.yaml)
		privateAuth.GET("/roles", roleCtrl.GetAllRoles)
		privateAuth.GET("/roles/:id", roleCtrl.GetRoleByID)
		privateAuth.POST("/roles", roleCtrl.CreateRole)
		privateAuth.PUT("/roles/:id", roleCtrl.UpdateRole)
		privateAuth.DELETE("/roles/:id", roleCtrl.DeleteRole)
	}

	return r
}
