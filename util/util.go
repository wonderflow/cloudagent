package util

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
)

func GetLocalIp() string {
	addrs, err := net.InterfaceAddrs()

	if err != nil {
		fmt.Printf("Get Local IP error: %v\n", err)
		os.Exit(1)
	}

	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

func Trap(cleanup func()) {
	c := make(chan os.Signal, 1)
	// we will handle INT, TERM, QUIT KILL here
	signals := []os.Signal{os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGQUIT}
	signal.Notify(c, signals...)

	interruptCount := uint32(0)
	for sig := range c {
		go func(sig os.Signal) {
			switch sig {
			case os.Interrupt, syscall.SIGTERM:
				if atomic.LoadUint32(&interruptCount) < 3 {
					// Initiate the cleanup only once
					if atomic.AddUint32(&interruptCount, 1) == 1 {
						// Call the provided cleanup handler
						cleanup()
						os.Exit(0)
					} else {
						return
					}
				} else {
					// 3 SIGTERM/INT signals received; force exit without cleanup
					fmt.Println("Forcing docker daemon shutdown without cleanup; 3 interrupts received")
				}
			case syscall.SIGQUIT, os.Kill:
				//DumpStacks()
				fmt.Println("Forcing docker daemon shutdown without cleanup on SIGQUIT")
			}
			//for the SIGINT/TERM, and SIGQUIT non-clean shutdown case, exit with 128 + signal #
			os.Exit(128 + int(sig.(syscall.Signal)))
		}(sig)
	}

}
