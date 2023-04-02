package utils

import (
	log "github.com/sirupsen/logrus"
	"net"
	"time"
)

const (
	ECHO_REQUEST_HEAD_LEN = 8
	ECHO_REPLY_HEAD_LEN   = 20
)

func Ping(host string, seq int16) (bool, int) {
	var (
		size    = 32
		timeout = 2000
		err     error
		success bool
	)
	// cname, _ := net.LookupCNAME(host)
	startTime := time.Now()
	conn, err := net.DialTimeout("ip4:icmp", host, time.Duration(timeout*1000*1000))
	if err != nil {
		log.Error(err)
		return false, 0
	}
	// ip := conn.RemoteAddr()
	// log.Info("ping " + cname + " [" + ip.String() + "]")

	id0, id1 := genidentifier(host)

	var msg []byte = make([]byte, size+ECHO_REQUEST_HEAD_LEN)
	msg[0] = 8                        // echo
	msg[1] = 0                        // code 0
	msg[2] = 0                        // checksum
	msg[3] = 0                        // checksum
	msg[4], msg[5] = id0, id1         //identifier[0] identifier[1]
	msg[6], msg[7] = gensequence(seq) //sequence[0], sequence[1]

	length := size + ECHO_REQUEST_HEAD_LEN

	check := checkSum(msg[0:length])
	msg[2] = byte(check >> 8)
	msg[3] = byte(check & 255)

	conn, err = net.DialTimeout("ip:icmp", host, time.Duration(timeout*1000*1000))

	if err != nil {
		log.Error(err)
		return false, 0
	}

	startTime = time.Now()
	_ = conn.SetDeadline(startTime.Add(time.Duration(timeout * 1000 * 1000)))
	_, err = conn.Write(msg[0:length])

	var receive = make([]byte, ECHO_REPLY_HEAD_LEN+length)
	_, err = conn.Read(receive)

	endDuration := int(int64(time.Since(startTime)) / (1000 * 1000))

	if err != nil || receive[ECHO_REPLY_HEAD_LEN+4] != msg[4] || receive[ECHO_REPLY_HEAD_LEN+5] != msg[5] || receive[ECHO_REPLY_HEAD_LEN+6] != msg[6] || receive[ECHO_REPLY_HEAD_LEN+7] != msg[7] || endDuration >= int(timeout) || receive[ECHO_REPLY_HEAD_LEN] == 11 {
		success = false
	} else {
		success = true
	}

	return success, endDuration
}

func checkSum(msg []byte) uint16 {
	sum := 0

	length := len(msg)
	for i := 0; i < length-1; i += 2 {
		sum += int(msg[i])*256 + int(msg[i+1])
	}
	if length%2 == 1 {
		sum += int(msg[length-1]) * 256 // notice here, why *256?
	}

	sum = (sum >> 16) + (sum & 0xffff)
	sum += (sum >> 16)
	var answer uint16 = uint16(^sum)
	return answer
}

func gensequence(v int16) (byte, byte) {
	ret1 := byte(v >> 8)
	ret2 := byte(v & 255)
	return ret1, ret2
}

func genidentifier(host string) (byte, byte) {
	return host[0], host[1]
}
