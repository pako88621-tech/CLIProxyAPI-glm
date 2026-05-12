# Specifiche dei Modelli GLM per Z.ai

Questa documentazione definisce le specifiche esatte da configurare nel file `internal/registry/models/models.json` affinché CLIProxyAPI possa gestire correttamente i modelli della famiglia GLM (Zhipu AI).

## 1. Modelli GLM Supportati

Basandosi sulla piattaforma Z.ai e sulle specifiche rilasciate da Zhipu AI, i modelli principali da mappare sono:

### GLM-5.1
- **Descrizione:** L'ultima versione di punta, ottimizzata per contesti ultra-lunghi, ragionamento complesso e compiti agentici.
- **Context Window:** 1.000.000 token (su Z.ai web tipicamente limitato a 128k o 200k in base al piano/guest).
- **Max Output Tokens:** 8192 (da verificare e mappare correttamente, raccomandato 8192).
- **Tool Calling:** Supporto completo.
- **Vision/Multimodale:** Sì.

### GLM-5
- **Descrizione:** Modello potente per ragionamento logico, matematica e programmazione, con capacità di "System 2 thinking" (o pensiero riflessivo).
- **Context Window:** 128.000 token.
- **Max Output Tokens:** 4096.
- **Tool Calling:** Supporto completo.

### GLM-4.7 / GLM-4.5
- **Descrizione:** Generazioni precedenti ottimizzate per efficienza e velocità, con versioni Air e Plus.
- **Context Window:** 128.000 token.
- **Max Output Tokens:** 4096.
- **Tool Calling:** Supportato per funzioni di base (come Web Search).

## 2. Integrazione nel JSON Registry

Il blocco da aggiornare in `models.json` dovrà rispettare il seguente formato:

```json
"zai": [
  {
    "id": "glm-5.1",
    "object": "model",
    "owned_by": "zhipu",
    "type": "zai",
    "display_name": "GLM-5.1",
    "context_length": 200000,
    "max_completion_tokens": 8192,
    "thinking": {
      "min": 1024,
      "max": 64000,
      "zero_allowed": true
    }
  },
  {
    "id": "glm-5",
    "object": "model",
    "owned_by": "zhipu",
    "type": "zai",
    "display_name": "GLM-5",
    "context_length": 128000,
    "max_completion_tokens": 4096,
    "thinking": {
      "min": 1024,
      "max": 32000,
      "zero_allowed": true
    }
  }
]
```

Questi limiti sono fondamentali per il router di CLIProxyAPI, in quanto evitano di troncare l'input dell'utente e informano il middleware su quanti token massimi di "thinking" possono essere spesi prima che il modello si fermi.
