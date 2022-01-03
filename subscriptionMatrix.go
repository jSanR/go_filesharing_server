package main

import "sync"

//Archivo que contiene la definición de una estructura con una matriz que contendrá las direcciones de los clientes
//suscritos por canal y con una variable mutex para cada matriz para evitar condiciones de carrera

type subscriptionsMatrix struct {
	arrMutex [NUMBER_OF_CHANNELS]sync.Mutex
	matrix   [NUMBER_OF_CHANNELS][]string
}

//Función que añade un nuevo cliente a la matriz
func (m *subscriptionsMatrix) append(address string, channel int8) {
	//Lock mutex
	m.arrMutex[channel-1].Lock()
	//Añadir el nuevo cliente al canal
	m.matrix[channel-1] = append(m.matrix[channel-1], address)
	//Unlock mutex
	m.arrMutex[channel-1].Unlock()
}

//Función que retorna los suscriptores de un canal
func (m *subscriptionsMatrix) readChannel(channel int8) []string {
	//Lock mutex
	m.arrMutex[channel-1].Lock()
	//Crear una copia del slice correspondiente
	var channelSubsCopy []string = make([]string, len(m.matrix[channel-1]))
	copy(channelSubsCopy, m.matrix[channel-1])
	defer m.arrMutex[channel-1].Unlock()
	return channelSubsCopy
}
