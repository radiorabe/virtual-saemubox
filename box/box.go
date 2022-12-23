/*
Package box contains the network and business logic of virtual-säemubox.

Copyright © 2020 Radio Bern RaBe - Lucas Bickel <hairmare@rabe.ch>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/
package box

import (
	"bufio"
	"fmt"
	"net"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	socketActive  bool
	socketPath    string
	socketPattern string
	targetMessage int32
)

func connectUDP(addr string) *net.UDPConn {
	udpAddr, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		log.Fatal(err)
	}

	localAddr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		log.Fatal(err)
	}

	conn, err := net.DialUDP("udp", localAddr, udpAddr)
	if err != nil {
		log.Fatal(err)
	}
	return conn
}

func connectTCP(addr string) net.Conn {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	return conn
}

func connectSocket(addr string) net.Conn {
	conn, err := net.Dial("unix", addr)
	if err != nil {
		log.Error(err)
	}
	return conn
}

func writeUDP(conn *net.UDPConn, value string) {
	log.Debugf("Writing to UDP connection '%s'", value)
	_, err := fmt.Fprintf(conn, "%s\r\n", value)
	if err != nil {
		log.Error(err)
	}
}

func writeTCP(conn net.Conn, value string) {
	log.Debugf("Writing to TCP connection: '%s'", value)
	_, err := fmt.Fprintf(conn, "%s\r\n", value)
	if err != nil {
		log.Error(err)
	}
}

func writeSock(conn net.Conn, value string) {
	log.Debugf("Writing to TCP connection: '%s'", value)
	_, err := conn.Write([]byte(value))
	if err != nil {
		log.Error(err)
	}
}

func onChange(klangbecken bool) {
	onair := "False"
	if klangbecken {
		log.Info("Starting Klangbecken")
		onair = "True"
	} else {
		log.Info("Stopping Klangbecken")
	}
	if socketActive {
		socket := connectSocket(socketPath)
		reader := bufio.NewReader(socket)

		writeSock(socket, fmt.Sprintf(socketPattern, onair))
		buffer, _, err := reader.ReadLine()
		if err != nil {
			log.Error(err)
		}
		log.Infof("Response from Liquidsoap '%s'", buffer)
		writeSock(socket, "quit\n")
		buffer, _, err = reader.ReadLine()
		if err != nil {
			log.Error(err)
		}
		log.Infof("Response from Liquidsoap '%s'", buffer)
		socket.Close()
	}
}

func waitAndRead(pathfinder net.Conn, target *net.UDPConn) {
	log.Info("Waiting for Pathfinder data.")

	reader := bufio.NewReader(pathfinder)

	defer pathfinder.Close()

	for {
		log.Debug("Reading from Pathfinder.")
		buffer, _, err := reader.ReadLine()
		if err != nil {
			log.WithError(err).Fatal("Failed to read from Pathfinder.")
		}
		trimmedData := trimmedStringFromBuffer(buffer)

		log.Infof("Received data '%s'", trimmedData)

		target, onChangeVal, err := checkTrimmedData(trimmedData)
		if err != nil {
			log.Fatal(err)
		}
		if target == 0 {
			continue
		}
		atomic.StoreInt32(&targetMessage, target)
		onChange(onChangeVal)

		log.Infof("Target message is now '%d'", atomic.LoadInt32(&targetMessage))
	}
}

func trimmedStringFromBuffer(buffer []byte) string {
	return strings.TrimRight(string(buffer), "\x00\r\n")
}

func checkTrimmedData(trimmedData string) (target int32, onChange bool, err error) {
	pinIsLow := regexp.MustCompile(`PinState=[lL]`)

	switch trimmedData {
	case "login successful":
		return 0, false, nil
	case "login failed":
		return 0, false, fmt.Errorf("Failed to login to Pathfinder.")
	}

	if pinIsLow.MatchString(trimmedData) {
		// Klangbecken
		return 1, true, nil
	}
	// Studio Live
	return 6, false, nil
}

// Execute initializes virtual-sämbox and runs is business logic.
func Execute(sendUDP bool, targetAddr string, pathfinderAddr string, pathfinderAuth string, device string, socket bool, socketPathOpt string, socketPatternOpt string) {

	socketActive = socket
	socketPath = socketPathOpt
	socketPattern = socketPatternOpt

	var target *net.UDPConn
	if sendUDP {
		log.Info("Connecting UDP...")
		target = connectUDP(targetAddr)
		log.Infof("Connected to target %s", targetAddr)
		defer target.Close()
	}
	pathfinder := connectTCP(pathfinderAddr)
	log.Infof("Connected to pathfinder %s", pathfinderAddr)

	go waitAndRead(pathfinder, target)

	writeTCP(pathfinder, fmt.Sprintf("LOGIN %s", pathfinderAuth))
	writeTCP(pathfinder, fmt.Sprintf("SUB %s", device))
	writeTCP(pathfinder, fmt.Sprintf("GET %s", device))

	for {
		if sendUDP {
			if atomic.LoadInt32(&targetMessage) != 0 {
				writeUDP(target, fmt.Sprintf("%d\r\n", atomic.LoadInt32(&targetMessage)))
			}
		}
		time.Sleep(600 * time.Millisecond)
	}
}
