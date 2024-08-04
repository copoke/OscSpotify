package main

import (
	"log"

	"github.com/hypebeast/go-osc/osc"
)

func main() {
	// Create a new OSC client to send messages to localhost on port 9000.
	// This is the default port for VRChat OSC messages.
	client := osc.NewClient("127.0.0.1", 9000)

	// Construct the OSC message with the target address.
	// This should match the parameter name in your VRChat avatar.
	message := osc.NewMessage("/avatar/parameters/slider")

	// Append a float value to the message. In this case, 0.2.
	message.Append(float32(0.2))

	// Send the message.
	if err := client.Send(message); err != nil {
		log.Fatalf("Error sending OSC message: %v", err)
	} else {
		log.Println("OSC message sent successfully")
	}
}
