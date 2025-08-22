package main

import (
	"net/http"
	"powerquery/db"
	"powerquery/query"

	"github.com/gin-gonic/gin"
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

	r := gin.Default()
	r.GET("/query", func(c *gin.Context) {
		username := c.Query("username")
		password := c.Query("password")
		cookies := c.Query("cookies")
		roomName := c.Query("roomName")

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
