package snat

import (
	"cds-k8s-tools/pkg/client"
	"cds-k8s-tools/pkg/config"
	"cds-k8s-tools/pkg/oscmd"
	"cds-k8s-tools/pkg/service"
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	NodeNameKey                = "NODE_NAME"
	AnnotationSNatKeyPre       = "snat.beta.kubernetes.io/"
	AnnotationSNatKeyFix       = "snat-ip"
	AnnotationSNatDefaultValue = "default"
	ConfigFileName             = "snat-config"
	ConfigDir                  = "/snat/"
	ConfigFmtType              = "ini"
	ConfigOnChangeFlag         = "TO_WATCH_CONFIG"
	RefreshInterval            = "snat.refresh.interval"
	CmdGetNetplanVersion       = "grep -i ubuntu /etc/issue"
	CmdGetNetplanGw            = "grep -HrEn '[^#]*gateway4' /etc/netplan/*.yaml"
	CmdResetNetplanNet         = "netplan apply"
	CmdGetIfcGw                = "grep -HrEn '[^#]*GATEWAY' /etc/sysconfig/network-scripts/ifcfg-eth* --exclude *.bak-2*"
	CmdResetIfcfgNet           = "ifup %s"
)

var (
	nodeName        string
	conf            *config.Conf
	refreshInterval int
	refreshCh       chan int
	sNatKey         string
)

func init() {
	nodeName = os.Getenv(NodeNameKey)
	if nodeName == "" {
		panic(fmt.Errorf("env value for (%s) blanked", NodeNameKey))
	}

	sNatKey = fmt.Sprintf("%s%s", AnnotationSNatKeyPre, AnnotationSNatKeyFix)

	configOnChange := false
	if c, err := strconv.ParseBool(os.Getenv(ConfigOnChangeFlag)); err == nil {
		configOnChange = c
	}
	conf = config.NewConf(ConfigFileName, ConfigFmtType, configOnChange, ConfigDir)
	refreshCh = make(chan int, 1)
}

func getNodeSNatRole() (sNatRole string) {
	v1node := client.Sa.GetNode(nodeName)
	if sip, found := v1node.Annotations[sNatKey]; found {
		if sip == "" {
			sNatRole = AnnotationSNatDefaultValue
		} else {
			sNatRole = sip
		}
	}
	return
}

func getGwFromConf() (sNatIP string) {
	forSNatRole := getNodeSNatRole()
	if forSNatRole == "" {
		log.Errorf("annotation key(%s) not found in node(%s)", sNatKey, nodeName)
		return
	}
	sNatRoleKey := fmt.Sprintf("default.%s%s", AnnotationSNatKeyPre, forSNatRole)
	sNatIP = conf.GetKeyString(sNatRoleKey)
	return
}

func getIntervalFromConf() int {
	refreshIntervalKey := fmt.Sprintf("default.%s", RefreshInterval)
	return conf.GetKeyInt(refreshIntervalKey)
}

func mastGetGwFromNode() (filename, line, gw string, isNetplan bool, err error) {
	for i := 0; i < 6; i++ {
		filename, line, gw, isNetplan, err = getGwFromNode()
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		} else {
			return
		}
	}
	return
}

func getGwFromNode() (filename, line, gw string, isNetplan bool, err error) {
	if _, err = oscmd.CmdToNode(CmdGetNetplanVersion); err == nil {
		isNetplan = true
	}

	// format: filename line pre gw
	var gwLine string
	if isNetplan {
		if gwLine, err = oscmd.CmdToNode(CmdGetNetplanGw); err != nil {
			return
		}
		// /etc/netplan/ifcfg-eth1.yaml:11:      gateway4: 10.240.238.4
		gwLineList := strings.Split(strings.TrimSpace(gwLine), ":")
		if len(gwLineList) < 4 {
			log.Infof("%v out %v", CmdGetNetplanGw, gwLine)
			return "", "", "", false, fmt.Errorf("getGwFromNode fail:%v out %v", CmdGetNetplanGw, gwLine)
		}

		filename = gwLineList[0]
		line = gwLineList[1]
		gw = strings.TrimSpace(gwLineList[3])
	} else {
		// for not ubuntu os
		if gwLine, err = oscmd.CmdToNode(CmdGetIfcGw); err != nil {
			return
		}
		// /etc/sysconfig/network-scripts/ifcfg-eth1:9:GATEWAY=10.240.197.8
		gwLineList := strings.Split(strings.TrimSpace(gwLine), ":")
		if len(gwLineList) < 3 {
			log.Infof("%v out %v", CmdGetIfcGw, gwLine)
			return "", "", "", false, fmt.Errorf("getGwFromNode fail:%v out %v", CmdGetIfcGw, gwLine)
		}
		filename = gwLineList[0]
		line = gwLineList[1]
		_gw := strings.Split(strings.TrimSpace(gwLineList[2]), "=")
		gw = strings.TrimSpace(_gw[1])
	}
	return
}

func UpdateGw() (err error) {
	var filename, line, gw string
	var isNetplan bool
	sNatIPNew := getGwFromConf()
	if sNatIPNew == "" {
		log.Warnf("snat-ip not found in config")
		return
	}
	filename, line, gw, isNetplan, err = mastGetGwFromNode()
	if err != nil {
		// 告警
		alarm(&service.AlarmMessage{
			NodeName: os.Getenv(NodeNameKey),
			Status:   "error",
			Msg:      err.Error(),
		})
		return nil
	}
	if sNatIPNew == gw {
		log.Infof("snat-ip(%s) was not changed", sNatIPNew)
		return
	}
	// 1. backup net file
	bakFilename := fmt.Sprintf("%s.bak%s", filename, time.Now().Format("-2006-01-02-15:04:05"))
	CmdBakFile := fmt.Sprintf("cp %s %s", filename, bakFilename)
	CmdBakFileBack := fmt.Sprintf("cp %s %s", bakFilename, filename)
	if _, err = oscmd.CmdToNode(CmdBakFile); err != nil {
		return
	}
	// 2. change filename:line:gw
	CmdUpdateGW := fmt.Sprintf("sed -i '%ss/%s/%s/' %s", line, gw, sNatIPNew, filename)
	if _, err = oscmd.CmdToNode(CmdUpdateGW); err != nil {
		// if update error, back filename
		_, err = oscmd.CmdToNode(CmdBakFileBack)
		return
	}
	// 3. reset network
	cmd := CmdResetNetplanNet
	if !isNetplan {
		nicName := strings.Split(filename, "ifcfg-")[1]
		cmd = fmt.Sprintf(CmdResetIfcfgNet, nicName)
	}
	if _, err = oscmd.CmdToNode(cmd); err != nil {
		// if reset error, back filename too
		_, err = oscmd.CmdToNode(CmdBakFileBack)
		return
	}
	// filenameBakDel := fmt.Sprintf("rm -r %s.bak-20*", filename)
	// _, err = oscmd.CmdToNode(filenameBakDel)
	return
}

func NewEventGw(name string) {
	if name != "timer" {
		SMonitor.ChangeMonitor()
	}
	log.Infof("starting event for update gw by %s", name)
	if err := UpdateGw(); err != nil {
		log.Fatal(err)
	}
	log.Infof("ended event for update gw by %s", name)
	if i := getIntervalFromConf(); i > 0 && i != refreshInterval {
		refreshInterval = i
		log.Infof("updated %s = %d", RefreshInterval, refreshInterval)
		refreshCh <- i
	} else if i == 0 {
		close(refreshCh)
	}
}

func Run() {
	NewEventGw(ConfigFileName)
	if refreshInterval == 0 {
		log.Fatalf("timer is not start, because %s = 0", RefreshInterval)
	}
	log.Infof("starting a timer(%ds)", refreshInterval)
	timer1 := time.NewTicker(time.Duration(refreshInterval) * time.Second)
	conf.OnConfChange(NewEventGw)
	conf.WatchConf()
	wg := new(sync.WaitGroup)
	wg.Add(4)
	go func() {
		defer wg.Done()
		for {
			select {
			case i, ok := <-refreshCh:
				if !ok || i == 0 {
					log.Infof("ending the timer")
					timer1.Stop()
					return
				}
				if i > 0 {
					log.Infof("updating the timer new interval %d", i)
					refreshInterval = i
					timer1.Reset(time.Duration(i) * time.Second)
				}
			case <-timer1.C:
				NewEventGw("timer")
			}
		}
	}()
	go CheckWorkerResult(wg)
	go CheckPodResult(wg)
	go CheckSNat(wg)
	wg.Wait()
}
