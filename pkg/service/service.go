package service

import (
	"bytes"
	"cds-k8s-tools/pkg/consts"
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

func Run() {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.Default()
	engine.Use(cors.Default())
	engine.Use(gin_zip.Gzip(gin_zip.DefaultCompression, gin_zip.WithDecompressFn(gin_zip.DefaultDecompressHandle)))

	engine.GET("/health", Health)
	engine.POST("/alarm", Alarm)
	engine.POST("/v2/alarm", AlarmV2)

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
	c.JSON(http.StatusOK, Success)
}

func Alarm(c *gin.Context) {
	var request AlarmReq
	err := c.ShouldBindJSON(&request)
	if err != nil {
		c.JSON(http.StatusBadRequest, ParamError)
	} else {
		//  上报 openapi 告警
		var cckAlarm = CckAlarmReq{
			ClusterId:  os.Getenv(consts.CDS_CLUSTER_ID),
			Site:       os.Getenv(consts.CDS_CLUSTER_REGION_ID),
			Msg:        request.Message.Msg,
			Hostname:   request.Name,
			AlarmType:  request.Type,
			AlarmGroup: request.Group,
			Ip:         request.Ip,
		}
		b, _ := json.Marshal(cckAlarm)

		req, _ := utils.NewCCKRequest(consts.SendAlarm, http.MethodPost, nil, bytes.NewReader(b))

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

func AlarmV2(c *gin.Context) {
	var request AlarmInstance
	err := c.ShouldBindJSON(&request)
	if err != nil {
		c.JSON(http.StatusBadRequest, ParamError)
	} else {
		AlarmChan <- request
	}
	c.JSON(http.StatusOK, Success)
}
