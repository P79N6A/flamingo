package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type TemplateHandler struct{}

func NewTemplateHandler() *TemplateHandler {
	return &TemplateHandler{}
}

func (handler *TemplateHandler) Register(e *gin.Engine) {
	group := e.Group("/templates")
	group.GET("/home", func(c *gin.Context) {
		c.HTML(http.StatusOK, "home.html", gin.H{})
	})
	group.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", gin.H{})
	})
	group.GET("/person", func(c *gin.Context) {
		cell := c.Query("cell")
		c.Set("cellphone", cell)
		c.HTML(http.StatusOK, "person.html", gin.H{})
	})
	group.GET("/signin_user", func(c *gin.Context) {
		c.HTML(http.StatusOK, "signin_user.html", gin.H{})
	})
	group.GET("/charge", func(c *gin.Context) {
		cellphone := c.Query("cell")
		if cellphone == "" {
			c.HTML(http.StatusOK, "person.html", gin.H{})
		} else {
			c.Set("cell", cellphone)
			c.HTML(http.StatusOK, "charge.html", gin.H{})
		}
	})
	// customer
	group.GET("/customer_home", func(c *gin.Context) {
		c.HTML(http.StatusOK, "customer_person.html", gin.H{})
	})
	group.GET("/customer_login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login_user.html", gin.H{})
	})
}
