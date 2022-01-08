package main

//Archivo con funciones de apoyo para el procesamiento de mensajes y solicitudes

import "encoding/binary"

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
