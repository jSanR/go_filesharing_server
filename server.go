package main

//Archivo con la función main del servidor.

import (
	"fmt"
	"net"
	"os"
)

//Constantes
const NUMBER_OF_CHANNELS = 8          //Cantidad de canales disponibles para que un cliente se suscriba
const BUFFER_SIZE = 1024              //Tamaño de buffer para recibir bytes del cliente en mensajes grandes
const LISTENER_PORT = "7101"          //Puerto sobre el que recibirá mensajes el servidor
const FILENAME_MAX_LENGTH = 40        //Tamaño máximo del nombre de un archivo que se recibe
const SEND_FILES_CONCURRENTLY = false //Determina si un archivo recibido se envía a los clientes de un canal de manera concurrente o secuencial

func main() {
	//Verificar argumentos (start PORT)
	if len(os.Args) != 2 || os.Args[1] != "start" {
		fmt.Println("Usage: server start\nExample: server start")
		os.Exit(0)
	}

	//Incicializar matriz que contendrá a los clientes conectados a cada canal
	var subsMatrix *subscriptionsMatrix = new(subscriptionsMatrix)

	//Iniciar servidor en localhost y el puerto específico
	var listener net.Listener
	var listenerError error
	listener, listenerError = net.Listen("tcp", "127.0.0.1:"+LISTENER_PORT)

	if listenerError != nil {
		fmt.Println("ERROR: Error while starting server" + listenerError.Error())
		return
	}

	fmt.Println("Server started on port " + LISTENER_PORT + ". Awaiting connections...")
	//Quedar a la espera de conexiones entrantes
	for {
		var connection net.Conn
		var connectionError error
		connection, connectionError = listener.Accept()

		if connectionError != nil {
			fmt.Println("ERROR: Error while accepting incoming connection: " + connectionError.Error())
			os.Exit(1)
		}

		//Interactuar con el cliente en otro goroutine
		go handleConnection(connection, subsMatrix)
	}

}
