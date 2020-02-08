/*
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
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"
	"regexp"
	"time"
)

var TargetMessage string

func connectUDP(addr string) *net.UDPConn {
	udpAddr, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		log.Fatal(err)
	}

	localAddr, err := net.ResolveUDPAddr("udp", ":0")

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

func waitAndRead(pathfinder net.Conn, target *net.UDPConn) {
	log.Info("Waiting for Pathfinder data.")

	buffer := make([]byte, 2048)
	pinIsLow := regexp.MustCompile(`PinState=[lL]`)

	defer pathfinder.Close()
	defer target.Close()

	for {
		log.Debug("Reading from Pathfinder.")
		_, err := pathfinder.Read(buffer)
		if err != nil {
			log.Errorf("Error '%s'", err)
		}
		log.Infof("Received data '%s'", buffer)

		if pinIsLow.Match(buffer) {
			// Klangbecken
			TargetMessage = "1"
		} else {
			// Studio Live
			TargetMessage = "6"
		}
	}
}

func Execute(targetAddr string, pathfinderAddr string, pathfinderAuth string, device string) {
	log.Info("Connecting...")
	target := connectUDP(targetAddr)
	log.Infof("Connected to target %s", targetAddr)
	pathfinder := connectTCP(pathfinderAddr)
	log.Infof("Connected to pathfinder %s", pathfinderAddr)

	go waitAndRead(pathfinder, target)

	writeTCP(pathfinder, fmt.Sprintf("LOGIN %s", pathfinderAuth))
	writeTCP(pathfinder, fmt.Sprintf("SUB %s", device))
	writeTCP(pathfinder, fmt.Sprintf("GET %s", device))

	for {
		if TargetMessage != "" {
			writeUDP(target, fmt.Sprintf("%s\r\n", TargetMessage))
		}
		time.Sleep(600 * time.Millisecond)
	}
}
