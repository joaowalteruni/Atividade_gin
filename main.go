package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Sala struct {
	ID         string   `json:"id"`
	Nome       string   `json:"nome"`
	Capacidade int      `json:"capacidade"`
	Recursos   []string `json:"recursos"`
}

type Aluno struct {
	ID    string `json:"id"`
	Nome  string `json:"nome"`
	Email string `json:"email"`
}

type Alocacao struct {
	SalaID        string `json:"sala_id"`
	DiaDaSemana   string `json:"dia_da_semana"`
	HorarioInicio string `json:"horario_inicio"`
	HorarioFim    string `json:"horario_fim"`
}

type Turma struct {
	ID         string    `json:"id"`
	Nome       string    `json:"nome"`
	Disciplina string    `json:"disciplina"`
	Professor  string    `json:"professor"`
	AlunosIDs  []string  `json:"alunos_matriculados"`
	Alocacao   *Alocacao `json:"alocacao,omitempty"`
}

var (
	dbSalas  = make(map[string]Sala)
	dbAlunos = make(map[string]Aluno)
	dbTurmas = make(map[string]Turma)
)

func main() {
	r := gin.New()
	r.Use(gin.Recovery())

	v1 := r.Group("/api/v1")
	{
		v1.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":    "healthy",
				"timestamp": time.Now(),
				"version":   "1.0.0",
			})
		})

		v1.POST("/salas", criarSala)
		v1.GET("/salas", listarSalas)

		v1.POST("/alunos", criarAluno)
		v1.GET("/alunos", listarAlunos)

		v1.POST("/turmas", criarTurma)
		v1.GET("/turmas", listarTurmas)
		v1.POST("/turmas/:id/matriculas", matricularAluno)
		v1.GET("/turmas/:id/matriculas", listarMatriculas)
		v1.POST("/turmas/:id/alocar", alocarSala)
	}

	r.Run(":8080")
}

func criarSala(c *gin.Context) {
	var sala Sala
	if err := c.ShouldBindJSON(&sala); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "Dados inválidos"})
		return
	}
	if sala.Capacidade <= 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": "Capacidade deve ser maior que zero"})
		return
	}
	dbSalas[sala.ID] = sala
	c.JSON(http.StatusCreated, sala)
}

func listarSalas(c *gin.Context) {
	var lista []Sala
	for _, s := range dbSalas {
		lista = append(lista, s)
	}
	c.JSON(http.StatusOK, lista)
}

func criarAluno(c *gin.Context) {
	var aluno Aluno
	if err := c.ShouldBindJSON(&aluno); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "Dados inválidos"})
		return
	}
	dbAlunos[aluno.ID] = aluno
	c.JSON(http.StatusCreated, aluno)
}

func listarAlunos(c *gin.Context) {
	var lista []Aluno
	for _, a := range dbAlunos {
		lista = append(lista, a)
	}
	c.JSON(http.StatusOK, lista)
}

func criarTurma(c *gin.Context) {
	var turma Turma
	if err := c.ShouldBindJSON(&turma); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "Dados inválidos"})
		return
	}
	turma.AlunosIDs = []string{}
	dbTurmas[turma.ID] = turma
	c.JSON(http.StatusCreated, turma)
}

func listarTurmas(c *gin.Context) {
	var lista []Turma
	for _, t := range dbTurmas {
		lista = append(lista, t)
	}
	c.JSON(http.StatusOK, lista)
}

func matricularAluno(c *gin.Context) {
	turmaID := c.Param("id")
	var req struct {
		AlunoID string `json:"aluno_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "Dados inválidos"})
		return
	}

	turma, turmaExiste := dbTurmas[turmaID]
	_, alunoExiste := dbAlunos[req.AlunoID]

	if !turmaExiste || !alunoExiste {
		c.JSON(http.StatusNotFound, gin.H{"erro": "Turma ou Aluno não encontrado"})
		return
	}

	for _, id := range turma.AlunosIDs {
		if id == req.AlunoID {
			c.JSON(http.StatusConflict, gin.H{"erro": "Aluno já matriculado nesta turma"})
			return
		}
	}

	if turma.Alocacao != nil {
		sala := dbSalas[turma.Alocacao.SalaID]
		if len(turma.AlunosIDs)+1 > sala.Capacidade {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": "Capacidade da sala excedida"})
			return
		}

		for _, t := range dbTurmas {
			if t.ID != turmaID && t.Alocacao != nil {
				matriculadoOutraTurma := false
				for _, id := range t.AlunosIDs {
					if id == req.AlunoID {
						matriculadoOutraTurma = true
						break
					}
				}

				if matriculadoOutraTurma {
					if turma.Alocacao.DiaDaSemana == t.Alocacao.DiaDaSemana {
						if turma.Alocacao.HorarioInicio < t.Alocacao.HorarioFim && turma.Alocacao.HorarioFim > t.Alocacao.HorarioInicio {
							c.JSON(http.StatusConflict, gin.H{"erro": "Conflito de agenda do aluno"})
							return
						}
					}
				}
			}
		}
	}

	turma.AlunosIDs = append(turma.AlunosIDs, req.AlunoID)
	dbTurmas[turmaID] = turma
	c.JSON(http.StatusOK, gin.H{"mensagem": "Aluno matriculado com sucesso"})
}

func listarMatriculas(c *gin.Context) {
	turmaID := c.Param("id")
	turma, existe := dbTurmas[turmaID]
	if !existe {
		c.JSON(http.StatusNotFound, gin.H{"erro": "Turma não encontrada"})
		return
	}

	var alunos []Aluno
	for _, alunoID := range turma.AlunosIDs {
		alunos = append(alunos, dbAlunos[alunoID])
	}
	c.JSON(http.StatusOK, alunos)
}

func alocarSala(c *gin.Context) {
	turmaID := c.Param("id")
	var req Alocacao
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"erro": "Dados inválidos"})
		return
	}

	turma, turmaExiste := dbTurmas[turmaID]
	sala, salaExiste := dbSalas[req.SalaID]

	if !turmaExiste || !salaExiste {
		c.JSON(http.StatusNotFound, gin.H{"erro": "Turma ou Sala não encontrada"})
		return
	}

	if sala.Capacidade < len(turma.AlunosIDs) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"erro": "Capacidade da sala menor que a quantidade de alunos matriculados"})
		return
	}

	for _, t := range dbTurmas {
		if t.ID != turmaID && t.Alocacao != nil {
			if t.Alocacao.SalaID == req.SalaID && t.Alocacao.DiaDaSemana == req.DiaDaSemana {
				if req.HorarioInicio < t.Alocacao.HorarioFim && req.HorarioFim > t.Alocacao.HorarioInicio {
					c.JSON(http.StatusConflict, gin.H{"erro": "Conflito de agenda: sala já alocada neste horário"})
					return
				}
			}
		}
	}

	turma.Alocacao = &req
	dbTurmas[turmaID] = turma
	c.JSON(http.StatusOK, gin.H{"mensagem": "Turma alocada com sucesso"})
}
