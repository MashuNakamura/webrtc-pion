# WebRTC-Pion

**WebRTC-Pion** is a Go-based project providing a WebRTC implementation for real-time camera streaming and screen sharing. This project serves as a reference or foundation for developing Real-Time Communication (RTC) applications using the Pion library.

---

## Features

- Real-time camera streaming
- Screen sharing capabilities
- Peer-to-peer connection management
- Optional debugging for stream event monitoring

---

## Scope & Limitations

Please note that this project is currently designed as a **proof of concept** to explore the capabilities of the [Pion WebRTC](https://github.com/pion/webrtc) library.

- **Self-View Only:** The current implementation functions as a **loopback** system. The media stream (video/screen) is sent to the server and reflected back to the same client immediately.
- **Single Session:** It is designed for testing and learning purposes and does not currently support multi-user video conferencing or complex routing architectures (like SFU or MCU).

---

## Installation

1. **Clone the repository**

   ```bash
   git clone https://github.com/username/webrtc-pemjar.git
   cd webrtc-pemjar
   ```

2. **Install dependencies**
   Ensure you have Go installed on your machine.

   ```bash
   go mod tidy
   ```

---

## Usage

Run the application using the following command:

```bash
go run .
```

Or strictly using the main file:

```bash
go run main.go
```

The application will start on the port defined in your configuration (usually http://localhost:8080).

---

## Debugging

Debug features are available in `webrtc.go` (specifically around lines 179 and 186):

```go
// fmt.Printf("[STREAM-END] Camera Stopped (ID: %s)\n", track.ID())
// fmt.Printf("[STREAM-END] Screen Sharing Stopped (ID: %s)\n", track.ID())
```
These logs are currently disabled (commented out) to prevent console spam during normal operation (due to the blocking nature of ReadRTP). If you wish to enable detailed stream debugging or handle specific stop events, simply uncomment these lines.

---

## Project Structure

- `main.go` : Application entry point and HTTP server configuration.
- `webrtc.go` : Core WebRTC implementation logic (Signaling, Track Handling, ICE).
- `struct.go` : Data structures for Track Handling.
- `README.md` : Project documentation.

---

## Credits

This project utilizes the [Pion WebRTC library](https://github.com/pion/webrtc) for the Go implementation.
