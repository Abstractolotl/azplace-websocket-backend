package main

import (
	"errors"
	"net"
)

func remove[Type comparable](slice []Type, element Type) []Type {
	for i, e := range slice {
		if e == element {
			return append(slice[:i], slice[i+1:]...)
		}
	}

	return slice
}

func getLocalIP() (string, error) {
	interfaces, err := net.InterfaceAddrs()

	if err != nil {
		return "", err
	}

	for _, address := range interfaces {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}

	return "", errors.New("could not find ip address")
}
