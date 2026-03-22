package router

import (
	"ticket/controller"
	"ticket/utils"
	midddleware "ticket/utils"

	"github.com/gin-gonic/gin"
)

var (
	r *gin.Engine
)

func InitRouter() {
	r = gin.Default()

	// 用户及角色接口
	userGroup := r.Group("/api/v1/users")
	{
		// 公开接口
		publicGroup := userGroup.Group("")
		{
			publicGroup.POST("", controller.Register)
			publicGroup.POST("/login", controller.Login)
		}

		// 需要登录的接口
		privateGroup := userGroup.Group("")
		privateGroup.Use(midddleware.JWTAuth())
		{
			privateGroup.GET("", controller.GetMyInfo)
			privateGroup.GET("/:id", controller.GetUserInfoByID)
			privateGroup.GET("/activities", controller.GetMyActivities)
			privateGroup.GET("/orders", controller.GetMyOrders)
			privateGroup.GET("/tickets", controller.GetMyTickets)
		}
	}

	roleGroup := r.Group("/api/v1/roles")
	roleGroup.Use(utils.AdminAuth())
	{
		roleGroup.GET("", controller.GetRoles)
	}

	// 活动接口
	activityGroup := r.Group("/api/v1/activities")
	{
		// 登录接口
		privateGroup := activityGroup.Group("")
		privateGroup.Use(utils.JWTAuth())
		{
			activityGroup.GET("", controller.GetActivities)
			activityGroup.GET("/:id", controller.GetActivityDetail)
		}

		// 管理员接口
		adminGroup := activityGroup.Group("")
		adminGroup.Use(utils.AdminAuth())
		{
			adminGroup.POST("", controller.CreateActivity)
			adminGroup.PUT("/:id", controller.UpdateAllActivity)
			adminGroup.PATCH("/:id", controller.UpdatePartialActivity)
			adminGroup.DELETE("/:id", controller.DeleteActivity)
		}
	}

	// // 订单接口
	// orderGroup := r.Group("/api/v1/orders")
	// {
	// 	// 管理员接口
	// 	adminGroup := orderGroup.Group("")
	// 	adminGroup.Use(utils.AdminAuth())
	// 	{
	// 		adminGroup.GET("", controller.GetOrders)
	// 	}

	// 	// 需要登录的接口
	// 	privateGroup := orderGroup.Group("")
	// 	privateGroup.Use(utils.JWTAuth())
	// 	{
	// 		privateGroup.POST("", controller.CreateOrder)
	// 		privateGroup.PATCH("/:id", controller.UpdateOrder)
	// 		privateGroup.GET("/:id", controller.GetOrderDetail)
	// 	}
	// }

	// 票接口
	ticketGroup := r.Group("/api/v1/tickets")
	{
		// 需要登录的接口
		privateGroup := ticketGroup.Group("")
		privateGroup.Use(utils.JWTAuth())
		{
			privateGroup.GET("/:id", controller.GetTicketDetail)
		}

		// 管理员接口
		adminGroup := ticketGroup.Group("")
		adminGroup.Use(utils.AdminAuth())
		{
			adminGroup.GET("/:id/verify", controller.VerifyTicket)
			adminGroup.PATCH("/:id/invalidate", controller.InvalidateTicket)
			adminGroup.GET("", controller.GetTickets)
			adminGroup.PATCH("/check", controller.VerifyTicketNO)
		}
	}
}

func GetRouter() *gin.Engine {
	return r
}
