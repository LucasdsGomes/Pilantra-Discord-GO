package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

// Bot estrutura para armazenar a sessão e métodos relacionados
type Bot struct {
	session *discordgo.Session
}

// NewBot inicializa uma nova instância do bot
func NewBot(token string) (*Bot, error) {
	sess, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	return &Bot{session: sess}, nil
}

// RegisterHandlers registra todos os comandos e eventos
func (bot *Bot) RegisterHandlers() {
	bot.session.AddHandler(bot.onMessageCreate)
}

// onMessageCreate lida com as mensagens recebidas
func (bot *Bot) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	switch {
	case m.Content == "!ping":
		s.ChannelMessageSend(m.ChannelID, "Pong!")
	case m.Content == "!help":
		bot.listCommands(s, m)
	case strings.HasPrefix(m.Content, "!choose"):
		bot.choose(s, m)
	case strings.HasPrefix(m.Content, "!clear"):
		bot.clear(s, m)
	case m.Content == "!whoistchola":
		bot.whoIsTchola(s, m)
	case strings.HasPrefix(m.Content, "!weather"):
		bot.weather(s, m)
	}

}

// listCommands exibe a lista de comandos disponíveis
func (bot *Bot) listCommands(s *discordgo.Session, m *discordgo.MessageCreate) {
	commands := `
**Comandos disponíveis:**
!ping - Pong!
!help - Lista os comandos disponíveis
!choose <opção1>, <opção2>,... - Escolhe uma das opções inseridas
!clear <quantidade> - Limpa o canal conforme a quantidade de mensagens inseridas
!whoistchola - Exibe um usuário do dia que é tchola
!weather <cidade> - Exibe o clima de uma cidade
`
	s.ChannelMessageSend(m.ChannelID, commands)
}

// choose escolhe uma opção dentre as quais o usuário inserir
func (bot *Bot) choose(s *discordgo.Session, m *discordgo.MessageCreate) {
	args := strings.Split(m.Content[8:], ",")
	var optionsList []string

	for _, option := range args {
		trimOption := strings.TrimSpace(option)
		if trimOption != "" {
			optionsList = append(optionsList, trimOption)
		}
	}
	
	if len(optionsList) < 2 {
		s.ChannelMessageSend(m.ChannelID, "Por favor, informe pelo menos duas opções.")
		return
	}

	randomIndex := rand.Intn(len(optionsList))
	chosenOption := optionsList[randomIndex]

	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Eu escolho: %s", chosenOption))
}

// clear elimina mensagens do chat
func (bot *Bot) clear(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	args := strings.Fields(m.Content)

	if len(args) < 2 {
		s.ChannelMessageSend(m.ChannelID, "Por favor, informe a quantidade de mensagens a serem limpas. Exemplo: !clear 10")
		return
	}

	count, err := strconv.Atoi(args[1])
	if err != nil || count < 1 || count > 100 {
		s.ChannelMessageSend(m.ChannelID, "Por favor, informe um número válido para a quantidade de mensagens a serem limpas. (1 a 100)")
		return
	}

	messages, err := s.ChannelMessages(m.ChannelID, count, "", "", "")
	if err != nil {
		fmt.Printf("Não foi possível acessar as mensagens: %v", err)
		return
	}

	for _, msg := range messages {
		err := s.ChannelMessageDelete(m.ChannelID, msg.ID)
		if err != nil {
			fmt.Printf("Não foi possível excluir a mensagem com ID %s.", msg.ID)
		}
	}

	// Envia a mensagem de confirmação
	msg, err := s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%d mensagens foram limpas.", count))
	if err != nil {
		fmt.Printf("Não foi possível enviar a confirmação: %v", err)
		return
	}

	// Cria um timer para aguardar 3 segundos
	timer := time.NewTimer(3 * time.Second)
	<-timer.C

	// Exclui a mensagem de confirmação após 3 segundos
	err = s.ChannelMessageDelete(m.ChannelID, msg.ID)
	if err != nil {
		fmt.Printf("Não foi possível excluir corretamente: %v", err)
	}
}




// whoIsTchola seleciona uma pessoa do servidor que é a tchola da vez!
func (bot *Bot) whoIsTchola(s *discordgo.Session, m *discordgo.MessageCreate) {
	guildID := m.GuildID
	usersServer, err := s.GuildMembers(guildID, "", 200)
	if err != nil {
		fmt.Printf("Não foi possível listar os membros: %v", err)
		return
	}

	if len(usersServer) == 0 {
		s.ChannelMessageSend(m.ChannelID, "Não há membros disponíveis no servidor.")
		return
	}

	var memberList []string
	for _, user := range usersServer {
		memberList = append(memberList, user.User.Username)
	}

	randomIndex := rand.Intn(len(memberList))
	randomName := memberList[randomIndex]

	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("O Tchola da vez é o(a): %s", randomName))
}


// weather diz o clima de uma cidade
// weather diz o clima de uma cidade
func (bot *Bot) weather(s *discordgo.Session, m *discordgo.MessageCreate) {
	args := strings.Fields(m.Content)
	if len(args) < 2 {
		s.ChannelMessageSend(m.ChannelID, "Por favor, informe a cidade. Exemplo: !weather São Paulo")
		return
	}

	city := strings.Join(args[1:], " ")
	apiKey := os.Getenv("WEATHER_API")

	// Codifica a cidade para a URL
	city = url.QueryEscape(city)

	url := fmt.Sprintf("http://api.openweathermap.org/data/2.5/weather?q=%s&appid=%s&units=metric", city, apiKey)

	resp, err := http.Get(url)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Não foi possível acessar a API do OpenWeatherMap: %v", err))
		return
	}
	defer resp.Body.Close()

	// Verifica o status da resposta
	if resp.StatusCode != http.StatusOK {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Erro na resposta da API: %s", resp.Status))
		return
	}

	var data struct {
		Weather []struct {
			Description string `json:"description"`
		} `json:"weather"`
		Main struct {
			Temp float64 `json:"temp"`
		} `json:"main"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		fmt.Printf("Erro ao processar a resposta da API: %v", err)
		return
	}

	if len(data.Weather) == 0 {
		fmt.Printf("Não foi possível obter a descrição do clima.")
		return
	}

	weatherDesc := data.Weather[0].Description
	temp := data.Main.Temp

	message := fmt.Sprintf("O clima em %s é %s com temperatura de %.1f°C.", city, weatherDesc, temp)
	s.ChannelMessageSend(m.ChannelID, message)
}

// Start inicia a sessão do bot
func (bot *Bot) Start() error {
	bot.session.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsGuildMembers |
		discordgo.IntentsGuildPresences

	err := bot.session.Open()
	if err != nil {
		return err
	}

	fmt.Println("Bot Online")
	return nil
}

// Stop finaliza a sessão do bot
func (bot *Bot) Stop() {
	bot.session.Close()
}

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
	
	bot, err := NewBot(os.Getenv("TOKEN"))
	if err != nil {
		log.Fatal("Erro ao criar o bot:", err)
	}

	// Registra os handlers
	bot.RegisterHandlers()

	// Inicia o bot
	err = bot.Start()
	if err != nil {
		log.Fatal("Erro ao iniciar o bot:", err)
	}

	// Captura sinal de interrupção para finalizar o bot com segurança
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Finaliza o bot
	bot.Stop()
}
