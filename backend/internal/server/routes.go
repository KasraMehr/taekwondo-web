package server

import "github.com/gin-gonic/gin"

func (s *Server) HelloWorldHandler(c *gin.Context) { c.JSON(200, gin.H{"message": "Hello World"}) }
