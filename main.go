package main

import (
	"context"
	"net/http"
	"powerquery/db"
	"powerquery/query"

	"github.com/gin-gonic/gin"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/viper"
)

type Config struct {
	RodUrl   string `mapstructure:"rod_url"`
	Database string `mapstructure:"database"`
}

func main() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}

	cache, err := db.NewBadgerCache(viper.GetString("database"))
	if err != nil {
		panic(err)
	}

	queryer, err := query.NewRodQueryer(cache, viper.GetString("rod_url"))
	if err != nil {
		panic(err)
	}

	s := server.NewMCPServer(
		"清水河电费查询",
		"0.0.1",
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	queryTool := mcp.NewTool("query",
		mcp.WithDescription("Query power (kWh) and balance (CNY) by room name, username and password. Leave username and password blank to use cached cookies if available."),
		mcp.WithString("room_name",
			mcp.Required(),
			mcp.Description("room name to query, usually 6 or 8 digits."),
		),
		mcp.WithString("username",
			mcp.Description("username to authenticate"),
		),
		mcp.WithString("password",
			mcp.Description("password to authenticate"),
		),
	)

	s.AddTool(queryTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		roomName, err := request.RequireString("room_name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		username := request.GetString("username", "")
		password := request.GetString("password", "")

		req := query.QueryRequest{
			Username: username,
			Password: password,
			Cookies:  "",
			RoomName: roomName,
		}
		result, err := queryer.DoQuery(req)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultStructuredOnly(result), nil
	})

	httpServer := server.NewStreamableHTTPServer(s)
	go httpServer.Start(":8081")

	r := gin.Default()
	r.GET("/query", func(c *gin.Context) {
		username := c.Query("username")
		password := c.Query("password")
		cookies := c.Query("cookies")
		roomName := c.Query("room_name")

		req := query.QueryRequest{
			Username: username,
			Password: password,
			Cookies:  cookies,
			RoomName: roomName,
		}

		resp, err := queryer.DoQuery(req)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, resp)
	})
	r.POST("/query", func(c *gin.Context) {
		var req query.QueryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		resp, err := queryer.DoQuery(req)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, resp)
	})
	r.Run(":8080")
}
