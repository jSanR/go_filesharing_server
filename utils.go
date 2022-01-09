package main

//Archivo con funciones de apoyo para el procesamiento de mensajes y solicitudes

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
)

//Función que crea un mensaje con la estructura estándar del protocolo
func createSimpleMessage(command int8, channel int8, body []byte) []byte {
	//Variables para el mensaje y cada una de sus partes
	var message []byte
	//Convertir el comando y añadirlo al mensaje
	message = append(message, byte(command))
	//Convertir el canal y añadirlo al mensaje
	message = append(message, byte(channel))
	//Obtener la longitud del contenido del mensaje
	var contentLength int64 = int64(len(body))
	var lengthBuffer []byte = make([]byte, 8)
	binary.LittleEndian.PutUint64(lengthBuffer, uint64(contentLength))
	//Añadir la longitud al mensaje
	message = append(message, lengthBuffer...)
	//Añadir el cuerpo o contenido al mensaje
	message = append(message, body...)

	//Retornar el mensaje ya lleno
	return message
}

//Función que procesa un mensaje (exceptuando el comando) relacionado con una suscripción de un cliente
func processSubscriptionMessage(connection net.Conn) (returnChannel int8, returnAddress string, returnStatus int) {
	var channelBuffer []byte = make([]byte, 1) //Buffer que recibe el canal de la suscripción
	var lengthBuffer []byte = make([]byte, 8)  //Buffer que recibe la longitud del contenido (en este caso la dirección del cliente)
	var contentBuffer []byte
	//Leer el canal al que el cliente se desea suscribir
	_, channelError := connection.Read(channelBuffer)
	//Error check
	if channelError != nil {
		fmt.Println("ERROR: Error while reading client's selected channel: " + channelError.Error())
		_, err := connection.Write(createSimpleMessage(3, 0, []byte("channel read error")))
		if err != nil {
			fmt.Println("ERROR: Error while sending response to client: " + err.Error())
		}
		return -1, "", 2
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
		return -1, "", 2
	}
	//Parsear el canal recibido
	var channel int8
	channel = int8(channelBuffer[0])
	//Comprobar que el canal recibido sea válido
	if channel < 1 || channel > NUMBER_OF_CHANNELS {
		fmt.Println("ERROR: The client's message specified an invalid channel")
		_, err := connection.Write(createSimpleMessage(3, 0, []byte("invalid channel (allowed channels: 1-"+strconv.Itoa(NUMBER_OF_CHANNELS)+")")))
		if err != nil {
			fmt.Println("ERROR: Error while sending response to client: " + err.Error())
		}
		return -1, "", 2
	}
	//Parsear la longitud del contenido
	var contentLength int64
	contentLength = int64(binary.LittleEndian.Uint64(lengthBuffer))
	//Comprobar que la longitud sea válida
	if contentLength <= 0 {
		fmt.Println("ERROR: The client's message specified an invalid content length")
		_, err := connection.Write(createSimpleMessage(3, 0, []byte("invalid content length")))
		if err != nil {
			fmt.Println("ERROR: Error while sending response to client: " + err.Error())
		}
		return -1, "", 3
	}
	//Leer el contenido del mensaje (dirección del cliente: IP + PORT)
	contentBuffer = make([]byte, contentLength)
	n, contentError := connection.Read(contentBuffer)
	//Error check
	if contentError != nil {
		fmt.Println("ERROR: Error while reading message's content: " + contentError.Error())
		_, err := connection.Write(createSimpleMessage(3, 0, []byte("content read error")))
		if err != nil {
			fmt.Println("ERROR: Error while sending response to client: " + err.Error())
		}
		return -1, "", 2
	}
	if int64(n) != contentLength {
		fmt.Printf("ERROR: Could not read content completely (expected: %d, real: %d)\n", contentLength, n)
		_, err := connection.Write(createSimpleMessage(3, 0, []byte("content incomplete read")))
		if err != nil {
			fmt.Println("ERROR: Error while sending response to client: " + err.Error())
		}
		return -1, "", 2
	}
	//Parsear el contenido
	var clientAddress string = string(contentBuffer)
	return channel, clientAddress, 0
}
