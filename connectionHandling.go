package main

//Archivo con funciones relacionadas con el manejo de conexiones entrantes al servidor

import (
	"encoding/binary"
	"fmt"
	"net"
	"strings"
)

func handleConnection(connection net.Conn, subsMatrix *subscriptionsMatrix) {
	var exitStatus int = -1                    //Código que indica el resultado de procesar la conexión actual
	var commandBuffer []byte = make([]byte, 1) //Buffer que recibe el comando inicial
	//Leer el comando recibido (la idea es que sea uno de los permitidos en el protocolo)
	_, messageError := connection.Read(commandBuffer)
	//Error check
	if messageError != nil {
		fmt.Println("ERROR: Error while reading client's command: " + messageError.Error())
		exitStatus = 2
		fmt.Printf("Handled connection (status: %d)\n", exitStatus)
		return
	}

	//Parsear el comando recibido
	var command int8
	command = int8(commandBuffer[0])

	switch command {
	//Suscripción a canal
	case 0:
		fmt.Println("Command received: subscribe")
		exitStatus = processSubscription(connection, subsMatrix)
	//Envío de archivo
	case 1:
		fmt.Println("Command received: send")
		exitStatus = processFileSharing(connection, subsMatrix)
	default:
		connection.Write([]byte("invalid command"))
		connection.Close()
		exitStatus = 0
	}
	fmt.Printf("Handled connection (status: %d)\n", exitStatus)
}

func processSubscription(connection net.Conn, subsMatrix *subscriptionsMatrix) int {
	var channelBuffer []byte = make([]byte, 1) //Buffer que recibe el canal de la suscripción
	var lengthBuffer []byte = make([]byte, 8)  //Buffer que recibe la longitud del contenido (en este caso la dirección del cliente)
	var contentBuffer []byte
	//Cerrar la conexión al terminar
	defer connection.Close()
	//Leer el canal al que el cliente se desea suscribir
	_, channelError := connection.Read(channelBuffer)
	//Error check
	if channelError != nil {
		fmt.Println("ERROR: Error while reading client's selected channel: " + channelError.Error())
		connection.Write([]byte("channel read error"))
		return 2
	}
	//Leer la longitud del contenido en el mensaje
	_, lengthError := connection.Read(lengthBuffer)
	//Error check
	if lengthError != nil {
		fmt.Println("ERROR: Error while reading message's content length: " + lengthError.Error())
		connection.Write([]byte("length read error"))
		return 2
	}
	//Parsear el canal recibido
	var channel int8
	channel = int8(channelBuffer[0])
	//Comprobar que el canal recibido sea válido
	if channel < 1 || channel > NUMBER_OF_CHANNELS {
		fmt.Println("ERROR: The client's message specified an invalid channel")
		connection.Write([]byte("invalid channel"))
		return 2
	}
	//Parsear la longitud del contenido
	var contentLength int64
	contentLength = int64(binary.LittleEndian.Uint64(lengthBuffer))
	//Comprobar que la longitud sea válida
	if contentLength <= 0 {
		fmt.Println("ERROR: The client's message specified an invalid content length")
		connection.Write([]byte("invalid content length"))
		return 3
	}
	//Leer el contenido del mensaje (dirección del cliente: IP + PORT)
	contentBuffer = make([]byte, contentLength)
	n, contentError := connection.Read(contentBuffer)
	//Error check
	if contentError != nil {
		fmt.Println("ERROR: Error while reading message's content: " + contentError.Error())
		connection.Write([]byte("content read error"))
		return 2
	}
	if int64(n) != contentLength {
		fmt.Printf("ERROR: Could not read content completely (expected: %d, real: %d)\n", contentLength, n)
		connection.Write([]byte("content incomplete read"))
		return 2
	}
	//Parsear el contenido
	var clientAddress string = string(contentBuffer)
	//Añadir la nueva dirección a la matriz de suscripciones
	subsMatrix.append(clientAddress, channel)
	fmt.Printf("New client subscribed to channel %d (%v)\n", channel, clientAddress)
	//Retornar un mensaje al cliente
	connection.Write([]byte("success"))
	return 0
}

func processFileSharing(connection net.Conn, subsMatrix *subscriptionsMatrix) int {
	var channelBuffer []byte = make([]byte, 1) //Buffer que recibe el canal por el que se enviará el archivo
	var lengthBuffer []byte = make([]byte, 8)  //Buffer que recibe la longitud del contenido (nombre y contenido de archivo)
	var filenameBuffer []byte = make([]byte, FILENAME_MAX_LENGTH)
	var fileBuffer []byte
	//Cerrar la conexión al terminar
	defer connection.Close()
	//Leer el canal seleccionado por el cliente
	_, channelError := connection.Read(channelBuffer)
	//Error check
	if channelError != nil {
		fmt.Println("ERROR: Error while reading client's selected channel: " + channelError.Error())
		connection.Write([]byte("channel read error"))
		return 2
	}
	//Leer la longitud del contenido en el mensaje
	_, lengthError := connection.Read(lengthBuffer)
	//Error check
	if lengthError != nil {
		fmt.Println("ERROR: Error while reading message's content length: " + lengthError.Error())
		connection.Write([]byte("length read error"))
		return 2
	}
	//Leer el nombre del archivo
	_, filenameError := connection.Read(filenameBuffer)
	//Error check
	if filenameError != nil {
		fmt.Println("ERROR: Error while reading file name: " + filenameError.Error())
		connection.Write([]byte("filename read error"))
		return 2
	}
	//Parsear el canal recibido
	var channel int8
	channel = int8(channelBuffer[0])
	//Comprobar que el canal recibido sea válido
	if channel < 1 || channel > NUMBER_OF_CHANNELS {
		fmt.Println("ERROR: The client's message specified an invalid channel")
		connection.Write([]byte("invalid channel"))
		return 3
	}
	//Parsear la longitud del contenido
	var contentLength int64
	contentLength = int64(binary.LittleEndian.Uint64(lengthBuffer))
	//Comprobar que la longitud sea válida
	if contentLength <= FILENAME_MAX_LENGTH {
		fmt.Println("ERROR: The client's message specified an invalid content length")
		connection.Write([]byte("invalid content length"))
		return 3
	}
	//Parsear el nombre del archivo
	var filename string = strings.Split(string(filenameBuffer), "\x00")[0]
	//Comprobar que el nombre del archivo no esté vacío
	if len(filename) == 0 {
		fmt.Println("ERROR: The client's message specified an empty file name")
		connection.Write([]byte("empty filename"))
		return 3
	}
	//Leer el resto del mensaje (contenido del archivo)
	fileBuffer = make([]byte, contentLength-FILENAME_MAX_LENGTH)
	n, fileError := connection.Read(fileBuffer)
	//Error check
	if fileError != nil {
		fmt.Println("ERROR: Error while reading file content: " + fileError.Error())
		connection.Write([]byte("file read error"))
		return 2
	}
	if int64(n) != contentLength-FILENAME_MAX_LENGTH {
		fmt.Printf("ERROR: Could not read file content completely (expected: %d, real: %d)\n", contentLength-FILENAME_MAX_LENGTH, n)
		connection.Write([]byte("file incomplete read"))
		return 2
	}
	//El archivo se ha leído y se tiene en un buffer
	fmt.Printf("File received from client (%v, %d bytes)\n", filename, contentLength-FILENAME_MAX_LENGTH)
	connection.Write([]byte("received"))
	//El archivo se enviará a los clientes suscritos al canal. Primero se armará el mensaje (longitud, filename, file)
	var message []byte
	message = append(message, lengthBuffer...)
	message = append(message, filenameBuffer...)
	message = append(message, fileBuffer...)
	//Se debe obtener la lista actual de clientes suscritos al canal recibido
	var clientList []string = subsMatrix.readChannel(channel)
	//Iniciar envío de archivos a cada cliente suscrito
	fmt.Printf("Sending received file to clients subscribed to channel %d (%d clients):\n", channel, len(clientList))
	for i, clientAddress := range clientList {
		fmt.Printf("(%d/%d) Sending file to client %v...\n", i+1, len(clientList), clientAddress)
		if SEND_FILES_CONCURRENTLY {
			go sendFileToClient(message, clientAddress) //Envío concurrente
		} else {
			sendFileToClient(message, clientAddress) //Envío secuencial
		}
	}
	return 0
}

func sendFileToClient(message []byte, clientAddress string) {
	//Conectarse con el cliente en cuestión (que en teoría debería tener un listener en la dirección recibida)
	var connection net.Conn
	var connectionError error
	connection, connectionError = net.Dial("tcp", clientAddress)
	defer connection.Close()
	//Error check
	if connectionError != nil {
		fmt.Println("ERROR: Error while trying to connect to client " + clientAddress + ": " + connectionError.Error())
		return
	}

	//Enviar mensaje
	var messageError error
	_, messageError = connection.Write(message)
	//Error check
	if messageError != nil {
		fmt.Println("ERROR: Error while sending message to client: " + messageError.Error())
		return
	}
	//Esperar una respuesta del cliente
	var responseBuffer []byte = make([]byte, BUFFER_SIZE)
	var response string
	n, responseError := connection.Read(responseBuffer)
	//Error check
	if responseError != nil {
		fmt.Println("ERROR: Error while receiving client's response: " + responseError.Error())
		return
	}
	response = string(responseBuffer[:n])
	//Interpretar respuesta
	switch response {
	case "received":
		fmt.Println("Sent file to client", clientAddress, "successfully")
	default:
		fmt.Println("ERROR: Client error (" + response + ")")
	}
}
