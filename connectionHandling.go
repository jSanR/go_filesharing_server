package main

//Archivo con funciones relacionadas con el manejo de conexiones entrantes al servidor

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
)

//Función que maneja la recepción de comandos de los clientes, llamando las funciones correspondientes
func handleConnection(connection net.Conn, subsMatrix *subscriptionMatrix) {
	/*
		Comandos existentes:
		0: subscribe (solicitud de suscripción)
		1: send (solicitud de envío de archivo)
		2: notify-success (notificar recepción/procesamiento exitoso de mensaje, no válido en este contexto)
		3: notify-failure (notificar error durante recepción/procesamiento de mensaje, no válido en este contexto)
		4: unsubscribe (solicitud para cancelar suscripción)
	*/
	var exitStatus int = -1                    //Código que indica el resultado de procesar la conexión actual
	var commandBuffer []byte = make([]byte, 1) //Buffer que recibe el comando inicial
	//Leer el comando recibido (la idea es que sea uno de los permitidos en el protocolo)
	_, messageError := connection.Read(commandBuffer)
	//Error check
	if messageError != nil {
		fmt.Println("ERROR: Error while reading client's command: " + messageError.Error())
		_, err := connection.Write(createSimpleMessage(3, 0, []byte("command read error")))
		if err != nil {
			fmt.Println("ERROR: Error while sending response to client: " + err.Error())
		}
		exitStatus = 2
		fmt.Printf("Handled connection (status: %d)\n", exitStatus)
		return
	}

	//Parsear el comando recibido
	var command int8
	command = int8(commandBuffer[0])

	switch command {
	case 0:
		//Suscripción a canal
		fmt.Println("Command received: subscribe")
		exitStatus = processSubscription(connection, subsMatrix)
	case 1:
		//Envío de archivo
		fmt.Println("Command received: send")
		exitStatus = processFileSharing(connection, subsMatrix)
	case 4:
		//Cancelación de suscripción
		fmt.Println("Command received: unsubscribe")
		exitStatus = cancelSubscription(connection, subsMatrix)
	default:
		//Comando inválido
		fmt.Println("Received invalid command. Closing connection...")
		_, err := connection.Write(createSimpleMessage(3, 0, []byte("invalid command")))
		if err != nil {
			fmt.Println("ERROR: Error while sending response to client: " + err.Error())
		}
		connection.Close()
		exitStatus = 0
	}
	fmt.Printf("Handled connection (status: %d)\n", exitStatus)
}

//Función para procesar una solicitud de suscripción de un cliente a un canal
func processSubscription(connection net.Conn, subsMatrix *subscriptionMatrix) int {
	var channel int8
	var clientAddress string
	var processStatus int
	//Cerrar la conexión al terminar
	defer connection.Close()
	channel, clientAddress, processStatus = processSubscriptionMessage(connection)
	if processStatus != 0 {
		return processStatus
	}
	//Añadir la nueva dirección a la matriz de suscripciones
	subsMatrix.append(clientAddress, channel)
	fmt.Printf("New client subscribed to channel %d (%v)\n", channel, clientAddress)
	//Retornar un mensaje al cliente
	_, err := connection.Write(createSimpleMessage(2, channel, []byte("subscribed")))
	if err != nil {
		fmt.Println("ERROR: Error while sending response to client: " + err.Error())
		return 2
	}
	return 0
}

//Función para procesar una solicitud de cancelación de suscripción de un canal
func cancelSubscription(connection net.Conn, subsMatrix *subscriptionMatrix) int {
	var channel int8
	var clientAddress string
	var processStatus int
	//Cerrar la conexión al terminar
	defer connection.Close()
	channel, clientAddress, processStatus = processSubscriptionMessage(connection)
	if processStatus != 0 {
		return processStatus
	}
	//Añadir la nueva dirección a la matriz de suscripciones
	subsMatrix.removeSubscriptor(clientAddress, channel)
	fmt.Printf("Client %v unsubscribed from channel %d\n", clientAddress, channel)
	//Retornar un mensaje al cliente
	_, err := connection.Write(createSimpleMessage(2, channel, []byte("unsubscribed")))
	if err != nil {
		fmt.Println("ERROR: Error while sending response to client: " + err.Error())
		return 2
	}
	return 0
}

//Función para procesar una solicitud de envío de archivo de un cliente a un canal
func processFileSharing(connection net.Conn, subsMatrix *subscriptionMatrix) int {
	var channelBuffer []byte = make([]byte, 1)                    //Buffer que recibe el canal por el que se enviará el archivo
	var lengthBuffer []byte = make([]byte, 8)                     //Buffer que recibe la longitud del contenido (nombre y contenido de archivo)
	var filenameBuffer []byte = make([]byte, FILENAME_MAX_LENGTH) //Buffer que recibe el nombre del archivo
	var fileBuffer []byte                                         //Buffer que recibe el contenido del archivo
	var tempBuffer []byte                                         //Buffer que va leyendo el contenido del archivo en partes
	//Cerrar la conexión al terminar
	defer connection.Close()
	//Leer el canal seleccionado por el cliente
	_, channelError := connection.Read(channelBuffer)
	//Error check
	if channelError != nil {
		fmt.Println("ERROR: Error while reading client's selected channel: " + channelError.Error())
		_, err := connection.Write(createSimpleMessage(3, 0, []byte("channel read error")))
		if err != nil {
			fmt.Println("ERROR: Error while sending response to client: " + err.Error())
		}
		return 2
	}
	//Leer la longitud del contenido en el mensaje
	_, lengthError := connection.Read(lengthBuffer)
	//Error check
	if lengthError != nil {
		fmt.Println("ERROR: Error while reading message's content length: " + lengthError.Error())
		_, err := connection.Write(createSimpleMessage(3, 0, []byte("length read error")))
		if err != nil {
			fmt.Println("ERROR: Error while sending response to client: " + err.Error())
		}
		return 2
	}
	//Leer el nombre del archivo
	_, filenameError := connection.Read(filenameBuffer)
	//Error check
	if filenameError != nil {
		fmt.Println("ERROR: Error while reading file name: " + filenameError.Error())
		_, err := connection.Write(createSimpleMessage(3, 0, []byte("filename read error")))
		if err != nil {
			fmt.Println("ERROR: Error while sending response to client: " + err.Error())
		}
		return 2
	}
	//Parsear el canal recibido
	var channel int8
	channel = int8(channelBuffer[0])
	//Comprobar que el canal recibido sea válido
	if channel < 1 || channel > NUMBER_OF_CHANNELS {
		fmt.Println("ERROR: The client's message specified an invalid channel")
		_, err := connection.Write(createSimpleMessage(3, 0, []byte("invalid channel")))
		if err != nil {
			fmt.Println("ERROR: Error while sending response to client: " + err.Error())
		}
		return 3
	}
	//Parsear la longitud del contenido
	var contentLength int64
	contentLength = int64(binary.LittleEndian.Uint64(lengthBuffer))
	//Comprobar que la longitud sea válida
	if contentLength <= FILENAME_MAX_LENGTH {
		fmt.Println("ERROR: The client's message specified an invalid content length")
		_, err := connection.Write(createSimpleMessage(3, 0, []byte("invalid content length")))
		if err != nil {
			fmt.Println("ERROR: Error while sending response to client: " + err.Error())
		}
		return 3
	}
	//Parsear el nombre del archivo
	var filename string = strings.Split(string(filenameBuffer), "\x00")[0]
	//Comprobar que el nombre del archivo no esté vacío
	if len(filename) == 0 {
		fmt.Println("ERROR: The client's message specified an empty file name")
		_, err := connection.Write(createSimpleMessage(3, 0, []byte("empty filename")))
		if err != nil {
			fmt.Println("ERROR: Error while sending response to client: " + err.Error())
		}
		return 3
	}
	fmt.Printf("Receiving file \"%v\"...\n", filename)
	//Leer el resto del mensaje (contenido del archivo)
	fileBuffer = make([]byte, 0) //Este buffer empieza vacío, pues se le irá concatenando el contenido del temporal
	tempBuffer = make([]byte, BUFFER_SIZE)
	var readLength int64 = 0
	//Lectura por partes para evitar problemas con archivos grandes
	for {
		//Leer al buffer temporal
		n, fileError := connection.Read(tempBuffer)
		//Error check
		if fileError == io.EOF { //Se concluyó la lectura
			break
		} else if fileError != nil { //Hubo un error de otro tipo
			fmt.Println("ERROR: Error while reading file content: " + fileError.Error())
			_, err := connection.Write(createSimpleMessage(3, 0, []byte("file read error")))
			if err != nil {
				fmt.Println("ERROR: Error while sending response to client: " + err.Error())
			}
			return 2
		}
		//Añadir lo leído al buffer del archivo
		fileBuffer = append(fileBuffer, tempBuffer[:n]...)
		//Actualizar la longitud leída
		readLength += int64(n)
		//Si ya se leyó el archivo completamente, se sale del bucle
		if readLength == contentLength-FILENAME_MAX_LENGTH {
			break
		}
	}

	if readLength != contentLength-FILENAME_MAX_LENGTH {
		fmt.Printf("ERROR: Could not read file content completely (expected: %d, real: %d)\n", contentLength-FILENAME_MAX_LENGTH, readLength)
		_, err := connection.Write(createSimpleMessage(3, 0, []byte("file incomplete read")))
		if err != nil {
			fmt.Println("ERROR: Error while sending response to client: " + err.Error())
		}
		return 2
	}
	//El archivo se ha leído y se tiene en un buffer
	fmt.Printf("File received from client (%v, %d bytes)\n", filename, contentLength-FILENAME_MAX_LENGTH)
	//Comunicar que se recibió el archivo al cliente que lo envió
	_, err := connection.Write(createSimpleMessage(2, channel, []byte("received")))
	if err != nil {
		fmt.Println("ERROR: Error while sending response to client: " + err.Error())
		return 2
	}
	//El archivo se enviará a los clientes suscritos al canal. Primero se armará el mensaje (longitud, filename, file)
	var message []byte = createSimpleMessage(1, channel, append(filenameBuffer, fileBuffer...))
	//Se debe obtener la lista actual de clientes suscritos al canal recibido
	var clientList []string = subsMatrix.readChannel(channel)
	//Iniciar envío de archivos a cada cliente suscrito
	fmt.Printf("Sending received file to clients subscribed to channel %d (%d clients):\n", channel, len(clientList))
	for i, clientAddress := range clientList {
		fmt.Printf("(%d/%d) Sending file to client %v...\n", i+1, len(clientList), clientAddress)
		if SEND_FILES_CONCURRENTLY {
			go sendFileToClient(message, fileBuffer, clientAddress) //Envío concurrente
		} else {
			sendFileToClient(message, fileBuffer, clientAddress) //Envío secuencial
		}
	}
	return 0
}

//Función para el envío de un archivo a un cliente suscrito
func sendFileToClient(message []byte, fileBytes []byte, clientAddress string) {
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

	//Enviar mensaje (excepto el archivo como tal, pues este se enviará iterativamente)
	var messageError error
	_, messageError = connection.Write(message[:10+FILENAME_MAX_LENGTH])
	//Error check
	if messageError != nil {
		fmt.Println("ERROR: Error while sending message to client: " + messageError.Error())
		return
	}
	//Enviar el archivo iterativamente
	var tempBuffer []byte = make([]byte, BUFFER_SIZE)
	var fileReadBuffer *bytes.Buffer = bytes.NewBuffer(fileBytes)
	var sentLength int = 0
	for {
		//Leer del contenido del archivo al buffer temporal
		readBytes, readError := fileReadBuffer.Read(tempBuffer)
		if readError != nil {
			if readError == io.EOF {
				fmt.Printf("File read completely (sent %d bytes)\n", sentLength)
				break
			}
			fmt.Println("ERROR: Error while reading file buffer: " + readError.Error())
			return
		}
		//fmt.Printf("Read %d bytes | ", readBytes)
		//Enviar lo leído al cliente
		sentBytes, sendError := connection.Write(tempBuffer[:readBytes])
		if sendError != nil {
			fmt.Println("ERROR: Error while sending file contents: " + sendError.Error())
			return
		}
		//Actualizar la cantidad enviada
		sentLength += sentBytes
		//Comprobar que lo que se lee se esté enviando completamente
		if readBytes != sentBytes {
			fmt.Println("ERROR: File buffer was sent incompletely")
			os.Exit(2)
		}
		//fmt.Printf("Sent %d bytes\n", sentBytes)
	}
	//Asegurarse de que el archivo se envió completamente
	if sentLength != len(fileBytes) {
		fmt.Println("ERROR: File was sent incompletely")
		return
	}
	//Esperar una respuesta del cliente
	var headerBuffer []byte = make([]byte, 10)
	var command int8
	var contentLength int64
	_, headerError := connection.Read(headerBuffer)
	//Error check
	if headerError != nil {
		fmt.Println("ERROR: Error while receiving client's response header: " + headerError.Error())
		return
	}
	//Parsear header
	command = int8(headerBuffer[0])
	contentLength = int64(binary.LittleEndian.Uint64(headerBuffer[2:]))
	//Leer contenido del mensaje
	var contentBuffer []byte = make([]byte, contentLength)
	var content string
	_, contentError := connection.Read(contentBuffer)
	//Error check
	if contentError != nil {
		fmt.Println("ERROR: Error while receiving client's response content: " + contentError.Error())
		return
	}
	//Parsear contenido del mensaje
	content = string(contentBuffer)
	//Interpretar respuesta
	switch command {
	case 2:
		fmt.Println("Sent file to client", clientAddress, "successfully")
	case 3:
		fmt.Println("ERROR: Client error (" + content + ")")
	default:
		fmt.Println("ERROR: Invalid command received from client:", command)
	}
}
