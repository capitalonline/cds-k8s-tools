package service

import (
	"bytes"
	"cds-k8s-tools/pkg/utils"
	"encoding/json"
	"fmt"
	"github.com/gin-contrib/cors"
	gin_zip "github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

const (
	CDS_CLUSTER_ID        = "CDS_CLUSTER_ID"
	CDS_CLUSTER_REGION_ID = "CDS_CLUSTER_REGION_ID"
)

func Run() {
	gin.SetMode(gin.ReleaseMode)
	gin.DisableConsoleColor()
	engine := gin.Default()
	engine.Use(cors.Default())
	engine.Use(gin_zip.Gzip(gin_zip.DefaultCompression, gin_zip.WithDecompressFn(gin_zip.DefaultDecompressHandle)))

	engine.GET("/health", Health)
	engine.POST("/alarm", Alarm)

	ch := make(chan error)
	go InterruptHandler(ch)
	go func() {
		ch <- engine.Run(":80")
	}()
	log.Info("closed:", <-ch)
}

func InterruptHandler(ch chan<- error) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	terminateError := fmt.Errorf("%s", <-c)
	ch <- terminateError
}

func Health(c *gin.Context) {
	c.JSON(http.StatusOK, map[string]interface{}{"code": "Success", "msg": "成功"})
}

func Alarm(c *gin.Context) {
	var request AlarmReq
	err := c.ShouldBindJSON(&request)
	if err != nil {
		c.JSON(http.StatusOK, map[string]interface{}{"code": "ParamError", "msg": "参数错误"})
	} else {
		//  上报 openapi 告警
		var cckAlarm = CckAlarmReq{
			ClusterId:  os.Getenv(CDS_CLUSTER_ID),
			Site:       os.Getenv(CDS_CLUSTER_REGION_ID),
			Msg:        request.Message.Msg,
			Hostname:   request.Name,
			AlarmType:  request.Type,
			AlarmGroup: request.Group,
			Ip:         request.Ip,
		}
		b, _ := json.Marshal(cckAlarm)

		req, _ := utils.NewCCKRequest(utils.SendAlarm, http.MethodPost, nil, bytes.NewReader(b))

		response, err := utils.DoOpenApiRequest(req)
		if err != nil {
			c.JSON(http.StatusOK,
				ReturnCommonResp(AlarmServiceError, fmt.Sprintf("%s, %s", response.Status, err)),
			)
		}
		content, err := io.ReadAll(response.Body)
		if response.StatusCode >= 400 {
			c.JSON(http.StatusOK, ReturnCommonResp(AlarmServiceWarn, fmt.Sprintf("%s, %s", response.Status, string(content))))
		} else {
			if strings.Contains(string(content), "Success") {
				c.JSON(http.StatusOK, Success)
			} else {
				c.JSON(http.StatusOK,
					ReturnCommonResp(AlarmServiceWarn, fmt.Sprintf("%s, %s", response.Status, string(content))),
				)
			}
		}
	}
}
