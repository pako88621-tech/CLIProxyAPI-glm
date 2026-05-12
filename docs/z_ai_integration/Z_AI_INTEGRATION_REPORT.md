# Integrazione Z.ai (ChatGLM) - Report di Analisi

## 1. Panoramica del servizio
Z.ai è una piattaforma di chat AI basata sui modelli ChatGLM (come GLM-4, GLM-5). Fornisce funzionalità di chat simili ad altre interfacce web (come ChatGPT, Claude).

## 2. Architettura Web Z.ai
Dall'analisi del frontend (JS bundle):
- L'URL principale dell'API privata sembra essere `https://api.z.ai/`
- Endpoint identificati per le conversazioni includono:
  - `/chats/new` (POST)
  - `/chats/`
  - `/chats/[id]` (GET, DELETE, POST)
- Le richieste usano autenticazione JWT via Header: `Authorization: Bearer <token>`
- È probabile che supportino SSE (Server-Sent Events) per le risposte in streaming come gli altri provider.
- Altri endpoint notati: `https://audio.z.ai/`, `https://image.z.ai/`, `https://ocr.z.ai/` suggerendo capacità multimodali.

## 3. Strategia di integrazione in CLIProxyAPI
Il progetto `CLIProxyAPI` è scritto in Go e gestisce più provider. Per aggiungere Z.ai, è necessario un nuovo "Executor" e un nuovo flusso di "Auth".

### 3.1 Struttura dell'Executor (Runtime)
I file da creare in `internal/runtime/executor/`:
- `zai_executor.go`: Implementerà l'interfaccia dell'executor per gestire le richieste, convertire il formato unificato di CLIProxyAPI nel formato richiesto dalle API private di `api.z.ai`, e tradurre le risposte SSE in ritorno al formato OpenAI.

### 3.2 Struttura Auth
I file da creare in `internal/auth/zai/`:
- `zai.go`: Logica per il login/Device Flow o l'acquisizione manuale del Bearer token (es. estraendo dal browser o tramite oauth2 se supportato).
- `token.go`: Strutture per salvare il token (`ZaiTokenStorage`), simulando la struttura usata per `Kimi` o `Claude`.
- `zai_proxy.go` (opzionale): Eventuale proxy middleware.

### 3.3 Struttura Translator
I file da creare in `internal/translator/zai/`:
- Traduttori per Request (da formato OpenAI a Z.ai) e Response (da SSE Z.ai a OpenAI ChatCompletion stream).

*Nota importante*: Poiché l'API di `Z.ai` è chiusa (frontend), l'implementazione dipenderà strettamente da reverse engineering del payload JSON inviato a `/chats/new` (e simili) e dal formato della risposta SSE.

## 4. Passi successivi
Per un'implementazione completa:
1. Catturare il traffico reale (tramite browser DevTools) per `https://chat.z.ai` e analizzare esattamente i payload scambiati.
2. Comprendere il meccanismo di login (Email/Password, SMS, o OAuth) per automatizzare la generazione del token in `internal/auth`.
