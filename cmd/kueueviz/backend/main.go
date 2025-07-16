package main

import (
	"fmt"
	"log"
	"net/http"

	"kueueviz/handlers"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"

	_ "net/http/pprof"
)

func main() {
	viper.AutomaticEnv()
	// Start pprof server for profiling
	go func() {
		log.Println("Starting pprof server on :6060")
		if err := http.ListenAndServe("localhost:6060", nil); err != nil {
			log.Fatalf("Error starting pprof server: %v", err)
		}
	}()

	_, dynamicClient, err := createK8sClient()
	if err != nil {
		log.Fatalf("Error creating Kubernetes client: %v", err)
	}
	r := gin.New()
	r.Use(gin.Logger())

	if err := r.SetTrustedProxies(nil); err != nil {
		log.Fatalf("Error setting trusted proxies: %v", err)
	}

	handlers.InitializeWebSocketRoutes(r, dynamicClient)

	viper.SetDefault("KUEUEVIZ_PORT", "8080")
	if err := r.Run(fmt.Sprintf(":%s", viper.GetString("KUEUEVIZ_PORT"))); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
