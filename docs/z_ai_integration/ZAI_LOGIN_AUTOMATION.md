# Automazione dell'Autenticazione Z.ai

Al momento CLIProxyAPI per Z.ai richiede l'estrazione manuale del `Bearer Token` tramite ispezione del traffico web e salvataggio nel file di autenticazione. Per ottenere un'esperienza senza frizioni (es. login diretto dalla TUI di CLIProxyAPI o da riga di comando), il processo deve essere automatizzato.

## Strategia 1: Generazione Token Guest Dinamica (Headless / Fetching)

Se l'utente non vuole usare il proprio account, il sito genera un Guest Token.
- **Vulnerabilità sfruttata:** Il token viene emesso alla prima visita da un endpoint dedicato o iniettato nell'HTML/cookies.
- **Azione richiesta:** Scrivere in `internal/auth/zai/zai.go` una funzione che faccia una `GET` iniziale verso `https://chat.z.ai`, verifichi eventuali cookie di tracciamento e successivamente chiami l'endpoint di login guest (presumibilmente `POST https://chat.z.ai/api/v1/auths/guest` o legga il JWT dal dump dell'HTML in `window.__INITIAL_STATE__`).

## Strategia 2: OAuth / SMS Login Automation (Account Reale per GLM-5.1)

Per accedere a GLM-5.1 o a context window più grandi, l'account registrato è obbligatorio.
Poiché CLIProxyAPI supporta nativamente il Device Flow per Kimi e Claude, l'approccio ideale per Z.ai è:

1. **Reverse Engineering dell'Endpoint Login:**
   Intercettare la chiamata in cui l'utente immette l'OTP inviato via mail o telefono (es. `POST https://chat.z.ai/api/v1/auth/login/code`).
2. **Creazione TUI in CLIProxyAPI:**
   Estendere `internal/tui/auth_tab.go` per chiedere all'utente la propria Email, invocare l'API che spedisce il codice, chiedere all'utente il codice via TUI, e infine inviare la convalida per ottenere il token permanente.
3. **Persistenza:**
   Sfruttare la struct `ZaiTokenStorage` già creata per salvare nel file system (es. `auths/zai_account_1.json`) l'Access Token e il Refresh Token.
4. **Refresh Loop:**
   Registrare Z.ai in `sdk/cliproxy/auth/auto_refresh_loop.go` affinché, se il token sta per scadere (ogni 1-7 giorni), venga rinnovato in background chiamando `/auth/refresh` in modo totalmente trasparente.

## Implementazione Prossima (Fase 2.1)
Si suggerisce di partire supportando l'immissione manuale del token (via web o UI di CLIProxyAPI) in quanto i meccanismi anti-bot (es. Cloudflare o recaptcha) possono bloccare la generazione dinamica di Guest Token tramite `net/http` se la firma TLS (JA3) non coincide con quella di un browser. Se il bot viene bloccato, valutare l'uso della libreria uTLS (già inclusa nel progetto).
