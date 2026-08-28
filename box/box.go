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

	"go.uber.org/zap"
)

var (
	socketActive  bool
	socketPath    string
	socketPattern string
	targetMessage atomic.Int32
	sourceDevice  string
)

var lastMessage = atomic.Pointer[time.Time]{}

func connectUDP(log *zap.SugaredLogger, addr string) *net.UDPConn {
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

func connectTCP(log *zap.SugaredLogger, addr string) net.Conn {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	return conn
}

func connectSocket(log *zap.SugaredLogger, addr string) net.Conn {
	conn, err := net.Dial("unix", addr)
	if err != nil {
		log.Error(err)
	}
	return conn
}

func writeUDP(log *zap.SugaredLogger, conn *net.UDPConn, value string) {
	log.Debugf("Writing to UDP connection '%s'", value)
	_, err := fmt.Fprintf(conn, "%s\r\n", value)
	if err != nil {
		log.Error(err)
	}
}

func writeTCP(log *zap.SugaredLogger, conn net.Conn, value string) {
	log.Debugf("Writing to TCP connection: '%s'", value)
	_, err := fmt.Fprintf(conn, "%s\r\n", value)
	if err != nil {
		log.Error(err)
	}
}

func writeSock(log *zap.SugaredLogger, conn net.Conn, value string) {
	log.Debugf("Writing to TCP connection: '%s'", value)
	_, err := conn.Write([]byte(value))
	if err != nil {
		log.Error(err)
	}
}

func onChange(log *zap.SugaredLogger, klangbecken bool) {
	onair := "False"
	if klangbecken {
		log.Info("Starting Klangbecken")
		onair = "True"
	} else {
		log.Info("Stopping Klangbecken")
	}
	if socketActive {
		socket := connectSocket(log, socketPath)
		reader := bufio.NewReader(socket)

		writeSock(log, socket, fmt.Sprintf(socketPattern, onair))
		buffer, _, err := reader.ReadLine()
		if err != nil {
			log.Error(err)
		}
		log.Infof("Response from Liquidsoap '%s'", buffer)
		writeSock(log, socket, "quit\n")
		buffer, _, err = reader.ReadLine()
		if err != nil {
			log.Error(err)
		}
		log.Infof("Response from Liquidsoap '%s'", buffer)
		socket.Close()
	}
}

func waitAndRead(log *zap.SugaredLogger, pathfinder net.Conn, target *net.UDPConn) {
	log.Info("Waiting for Pathfinder data.")

	reader := bufio.NewReader(pathfinder)

	defer pathfinder.Close()

	for {
		log.Debug("Reading from Pathfinder.")
		buffer, _, err := reader.ReadLine()
		if err != nil {
			log.Fatal("Failed to read from Pathfinder.")
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
		lastMessage.Store(func() *time.Time {
			t := time.Now()
			return &t
		}())
		targetMessage.Store(target)
		onChange(log, onChangeVal)

		log.Infof("Target message is now '%d'", targetMessage.Load())
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

func resubPathfinder(log *zap.SugaredLogger, pathfinder net.Conn) (err error) {
	writeTCP(log, pathfinder, fmt.Sprintf("UNSUB %s", sourceDevice))
	writeTCP(log, pathfinder, fmt.Sprintf("SUB %s", sourceDevice))
	// ensure we didn't miss a message while we were unsubbed by generating one
	writeTCP(log, pathfinder, fmt.Sprintf("GET %s", sourceDevice))
	return nil
}

// Execute initializes virtual-sämbox and runs is business logic.
func Execute(log *zap.SugaredLogger, sendUDP bool, targetAddr string, pathfinderAddr string, pathfinderAuth string, device string, socket bool, socketPathOpt string, socketPatternOpt string) {

	socketActive = socket
	socketPath = socketPathOpt
	socketPattern = socketPatternOpt
	sourceDevice = device
	lastMessage.Store(func() *time.Time {
		t := time.Now()
		return &t
	}())

	var target *net.UDPConn
	if sendUDP {
		log.Info("Connecting UDP...")
		target = connectUDP(log, targetAddr)
		log.Infof("Connected to target %s", targetAddr)
		defer target.Close()
	}
	pathfinder := connectTCP(log, pathfinderAddr)
	log.Infof("Connected to pathfinder %s", pathfinderAddr)

	go waitAndRead(log, pathfinder, target)

	writeTCP(log, pathfinder, fmt.Sprintf("LOGIN %s", pathfinderAuth))
	writeTCP(log, pathfinder, fmt.Sprintf("SUB %s", sourceDevice))
	writeTCP(log, pathfinder, fmt.Sprintf("GET %s", sourceDevice))

	for {
		if sendUDP {
			if targetMessage.Load() != 0 {
				writeUDP(log, target, fmt.Sprintf("%d\r\n", targetMessage.Load()))
			}
		}
		// Resubscribe every then and now just in case pathfinder rebooted on us.
		// basically i'm giving up on trying to figure out how we can have stale
		// connections to pathfinder when it reboots. This shows up as us not being
		// subscribed to what we need anymore while the connection never gets
		// terminated properly.
		// I can't reprod the exact scenario because everything i try terminates the
		// the connection properly (except actually rebooting the device that is).
		// The code below works around all of this by resubscribing using UNSUB/SUB
		// again after some time has passed. We might consider this as renewing the
		// lease on the subscriptions.
		// to make sure we don't spam klangbecken with "started" messages that lead
		// to tracks being skipped, this logic is only triggered when we haven't
		// heard from pathfinder for a bit over an hour and we don't think klangbecken
		// is currently running.
		// I'm hoping that this will make it so that this becomes self-healing and
		// and does so as soon as we have a show that runs for more than
		if time.Since(*lastMessage.Load()) > time.Minute*63 && targetMessage.Load() != 1 {
			log.Info("Resubbing Pathfinder")
			err := resubPathfinder(log, pathfinder)
			if err != nil {
				log.Fatal(err)
			}
		}
		time.Sleep(600 * time.Millisecond)
	}
}
