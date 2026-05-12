# Integrazione delle Capacità Agentiche (Tool Calling) su Z.ai

Affinché i modelli GLM-5 fungano da veri "Agenti", non basta chattare; devono poter eseguire codice o strumenti esposti dal client (come Kilo Code).

Questa documentazione delinea le future implementazioni per la gestione dei Tools in `CLIProxyAPI` per il provider Z.ai.

## 1. Analisi Struttura "Tools" in Z.ai

Z.ai espone dei tool nativi chiamati "features" (es. web search, ppt-maker, mcp_servers). Per iniettare i *nostri* tools (quelli definiti dal client tramite lo standard OpenAI API o Claude API), dobbiamo mapparli nel payload del `chat/new`.

Dall'analisi del payload, Z.ai accetta un array `mcp_servers` o `tools`. Sebbene l'API interna web sia primariamente rivolta ai propri tool interni (es. plugin web search), per sbloccare l'esecuzione di tool *client-side* (Function Calling standard), l'esecutore dovrà iniettare i "System Prompt" o sfruttare un blocco nativo `functions` o `tools`.

## 2. Traduzione in `request.go` (Da OpenAI Tools a Z.ai)

Se un client (Kilo Code) invia:
```json
{
  "tools": [{
    "type": "function",
    "function": {
      "name": "bash",
      "description": "Executes a shell command."
    }
  }]
}
```

Dobbiamo scrivere in `internal/translator/zai/request.go` un blocco di parsing che legga questo array `tools`.
**Opzione A (Nativa se scoperta):** Z.ai accetta un campo `tools` uguale a quello di OpenAI. In tal caso, si fa un passaggio diretto (passthrough) nel body HTTP in `ZaiChatCreateRequest`.
**Opzione B (System Prompt Injection):** Se l'API privata web non supporta il function calling generico per ragioni di sicurezza, i tools verranno iniettati dinamicamente nel System Prompt, indicando a GLM di rispondere con un output XML formattato (`<tool_call><name>bash</name>...</tool_call>`), trasformandolo così in un finto tool-calling lato server, che viene poi riconvertito nella risposta.

## 3. Traduzione in `response.go` (Da Z.ai a Tool Call chunk)

Quando il modello decide di chiamare un tool, la stringa SSE in streaming potrebbe arrivare in vari modi.

Se usiamo l'Opzione B (Prompt Injection), lo stream conterrà testo puro.
- Il modulo `response.go` dovrà usare uno stato interno (Buffer/Regex) per intercettare l'apertura del tag `<tool_call>` (o analogo).
- Una volta riconosciuto, i Chunk emessi verso CLIProxyAPI devono cambiare il loro formato da `choices[0].delta.content` a `choices[0].delta.tool_calls`.

Questo processo (già in uso per altri provider che non supportano tools nativi, come Antigravity base) si chiama "Synthetic Tool Calling".

Se usiamo l'Opzione A (Nativa), Z.ai probabilmente invierà eventi SSE del tipo `{"phase": "tool_call", "tool_name": "bash", "tool_args": "{...}"}` che verranno semplicemente convertiti e inoltrati al client.

## 4. Traduzione dei Risultati (Tool Result)

Quando il client ha eseguito lo script in Bash, invierà la risposta in un nuovo messaggio:
```json
{"role": "tool", "tool_call_id": "call_abc123", "content": "Exit 0"}
```
`request.go` dovrà mappare questo `role: "tool"` in un messaggio utente con formato stringa se l'API privata non gestisce ruoli custom, oppure inoltrarlo esattamente se nativo.

## Conclusione

L'implementazione delle capacità agentiche per Z.ai richiederà test empirici (sottomettendo request JSON modificate via Postman/CURL a `https://chat.z.ai/api/v1/chats/new`) per determinare se `Zhipu` permette chiamate a function arbitrarie sull'endpoint web.
