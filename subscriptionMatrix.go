package main

//Archivo que contiene la definición de una estructura con una matriz que contendrá las direcciones de los clientes
//suscritos por canal y con una variable mutex para cada matriz para evitar condiciones de carrera

import (
	"sync"
	"time"
)

//Para cada canal existirá un mapa (las llaves serán las direcciones de los clientes y el valor será el tiempo de su suscripción)
type subscriptionMap map[string]time.Time

type subscriptionMatrix struct {
	arrMutex [NUMBER_OF_CHANNELS]sync.Mutex
	matrix   [NUMBER_OF_CHANNELS]subscriptionMap
}

//Función que retorna una nueva matriz (inicializando sus mapas)
func newSubscriptionMatrix() *subscriptionMatrix {
	var matrix *subscriptionMatrix = new(subscriptionMatrix)
	for i := 0; i < len(matrix.matrix); i++ {
		matrix.matrix[i] = make(subscriptionMap)
	}
	return matrix
}

//Función que añade un nuevo cliente a la matriz
func (m *subscriptionMatrix) append(address string, channel int8) {
	//Lock mutex
	m.arrMutex[channel-1].Lock()
	//Añadir el nuevo cliente al canal
	m.matrix[channel-1][address] = time.Now()
	//Unlock mutex
	m.arrMutex[channel-1].Unlock()
}

//Función que retorna los suscriptores de un canal
func (m *subscriptionMatrix) readChannel(channel int8) []string {
	//Lock mutex
	m.arrMutex[channel-1].Lock()
	//Crear una copia del las keys del mapa correspondiente
	var channelSubsCopy []string = make([]string, len(m.matrix[channel-1]))
	var i int = 0
	for address := range m.matrix[channel-1] {
		channelSubsCopy[i] = address
		i++
	}
	//Unlock mutex
	defer m.arrMutex[channel-1].Unlock()
	return channelSubsCopy
}

//Función que elimina un cliente de un determinado canal
func (m *subscriptionMatrix) removeSubscriptor(address string, channel int8) {
	//Lock mutex
	m.arrMutex[channel-1].Lock()
	//Retirar el cliente del mapa correspondiente al canal
	delete(m.matrix[channel-1], address)
	//Unlock mutex
	m.arrMutex[channel-1].Unlock()
}
