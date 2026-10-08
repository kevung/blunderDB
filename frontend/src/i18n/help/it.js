// GENERATED FILE — do not edit by hand, and do not translate it here.
//
// Produced by `go run ./cmd/help-gen` (make help) from:
//   - doc/source/manuel.rst      → the "manual" tab
//   - doc/source/raccourcis.rst  → the "shortcuts" tab
//   - doc/source/cmd_mode.rst    → the "commands" tab
//   - doc/source/locale/<lang>/LC_MESSAGES/*.po for the eight translations
//   - frontend/src/i18n/help/prose/<lang>.html → the "about" tab
//
// Fix the documentation (and its .po catalogues), or the prose fragment, then
// run `make help`. TestHelpBundlesAreCurrent fails if this file is stale.
export default {
    manual: `
<h3>Introduzione</h3>
<p>blunderDB è un software per creare database di posizioni di backgammon. Il suo punto di forza principale è fornire un luogo unico in cui aggregare le posizioni che un giocatore ha incontrato (online, in torneo) e poterle riesaminare filtrandole secondo vari filtri combinabili arbitrariamente. blunderDB può anche essere usato per creare cataloghi di posizioni di riferimento.</p>
<p>Le posizioni sono memorizzate in un database rappresentato da un file <em>.db</em>. L'applicazione desktop apre questo file direttamente, mai un indirizzo di rete: la modalità server è un'altra modalità dello stesso binario, e si passa dall'una all'altra esportando o migrando il database, non puntando l'applicazione verso un URL.</p>
<h3>Interazioni principali</h3>
<p>Le principali interazioni possibili con blunderDB sono:</p>
<ul>
<li>aggiungere una nuova posizione,</li>
<li>modificare una posizione esistente,</li>
<li>copiare l'immagine del board negli appunti (PNG) tramite <strong>CTRL-X</strong>, oppure con l'analisi completa tramite <strong>CTRL-X CTRL-X</strong>,</li>
<li>eliminare una posizione esistente,</li>
<li>cercare una o più posizioni,</li>
<li>importare match da diverse fonti (XG, GNUbg, BGBlitz, Jellyfish, HedgeHog), compresi i commenti dai file XG,</li>
<li>navigare tra le mosse di un match importato,</li>
<li>organizzare le posizioni in raccolte,</li>
<li>organizzare i match in tornei.</li>
</ul>
<p>L'utente può etichettare liberamente le posizioni con tag e annotarle tramite commenti.</p>
<h3>Descrizione dell'interfaccia</h3>
<p>L'interfaccia di blunderDB è composta, dall'alto verso il basso, da:</p>
<ul>
<li>[in alto] la barra degli strumenti, che raccoglie tutte le principali operazioni eseguibili sul database,</li>
<li>[al centro] l'area di visualizzazione principale, che permette di visualizzare o modificare posizioni di backgammon,</li>
<li>[in basso] la barra di stato, che presenta diverse informazioni sul database o sulla posizione corrente e integra la riga di comando.</li>
</ul>
<p>Possono essere visualizzati dei pannelli per:</p>
<ul>
<li>visualizzare i dati di analisi associati alla posizione corrente provenienti da eXtreme Gammon (XG), GNUbg o BGBlitz,</li>
<li>visualizzare, aggiungere o modificare commenti,</li>
<li>cercare e filtrare posizioni secondo criteri combinabili,</li>
<li>visualizzare e gestire le raccolte di posizioni (pannello raccolte),</li>
<li>visualizzare l'elenco dei match importati e navigare tra le mosse di un match (pannello match),</li>
<li>visualizzare e gestire i tornei (pannello tornei),</li>
<li>visualizzare le statistiche di performance (pannello Stats),</li>
<li>calcolare l'EPC (Effective Pip Count) di una posizione di bearoff (pannello Eval),</li>
<li>allenarsi su esercizi di calcolo (pannello Allenamento),</li>
<li>studiare le posizioni tramite ripetizione dilazionata (pannello Anki),</li>
<li>trascrivere un match a mano (pannello Trascrizione),</li>
<li>visualizzare i metadati del database (pannello Metadati).</li>
</ul>
<p>L'altezza del pannello si regola trascinandone la maniglia; resta la stessa da una scheda all'altra.</p>
<p>Possono comparire finestre modali per:</p>
<ul>
<li>visualizzare la guida di blunderDB, il cui campo di ricerca (<em>/</em> vi porta il cursore) seleziona le occorrenze del testo digitato, senza distinzione di maiuscole né di accenti (<em>INVIO</em> passa alla successiva, <em>MAIUSC-INVIO</em> alla precedente),</li>
<li>visualizzare il catalogo delle visite guidate (vedi Visite guidate e database di esempio),</li>
<li>configurare le impostazioni di esportazione del database,</li>
<li>configurare blunderDB, in particolare la lingua dell'interfaccia (vedi Configurazione).</li>
</ul>
<p>L'area di visualizzazione principale mette a disposizione dell'utente:</p>
<ul>
<li>un board per visualizzare o modificare una posizione di backgammon,</li>
<li>il livello e il proprietario del cubo,</li>
<li>il pip count di ciascun giocatore,</li>
<li>il punteggio di ciascun giocatore,</li>
<li>i dadi da giocare. Se sui dadi non è mostrato alcun valore, la posizione dei dadi indica quale giocatore ha il turno e che la posizione è una decisione di cubo. Quando la decisione di cubo è una risposta a un raddoppio (prendi/passa), il cubo proposto è mostrato al centro del board, al valore offerto.</li>
</ul>
<p>Un clic destro sulla dama apre un menu contestuale che propone: valutare la posizione visualizzata nel pannello Eval, valutarne lo specchio, copiare l'immagine della dama con la sua analisi negli appunti (l'equivalente di <em>CTRL-X CTRL-X</em>, meno facile da scoprire), <strong>salvare l'immagine in un file</strong> in SVG o PNG, aprire una nuova vista su questa posizione, e — se la posizione viene già dalla base — aggiungerla a un mazzo Anki (ripetizione dilazionata) oppure ordinare le sue <strong>posizioni vicine</strong> (vedere Pannello Ricerca). Durante una mossa giocata sulla scacchiera (quiz, Trascrizione), il menu inizia con <em>Ricomincia</em>, che azzera la mossa senza toccare la posizione. In modifica e in Eval, dove il clic destro sulla scacchiera posiziona le pedine, il menu si apre con un clic destro fuori dalla scacchiera — fuori da dadi, cubo e punteggi — e propone solo <em>Cancella la posizione</em> (come <em>BACKSPACE</em>) e <em>Posizione iniziale</em>, che dispone le pedine di una partita nuova.</p>
<p>Gli appunti sono il gesto corrente; salvare è l'altro bisogno — l'illustrazione di un articolo, di un messaggio di forum, di una lezione. L'<strong>SVG</strong> è proposto perché la dama lo è: è la forma che sopravvive a un ingrandimento, quella che si mette in un documento senza sfocarla. Il PNG ne deriva, come la copia negli appunti: un solo rendering, tre destinazioni, quindi nessuna può divergere dalle altre. Questo menu non compare nel pannello Eval né nel pannello Ricerca, dove il tasto destro serve già a posare le pedine dell'altro colore. Vedere Portare una posizione nel pannello Eval per portare una posizione nel pannello Eval.</p>
<p>La barra di stato è strutturata da sinistra a destra con le seguenti informazioni:</p>
<ul>
<li>la riga di comando, accessibile premendo il tasto <em>SPAZIO</em>,</li>
<li>un messaggio informativo relativo a un'operazione eseguita dall'utente,</li>
<li>l'indice della posizione corrente, seguito dal numero di posizioni nella biblioteca corrente (o le informazioni di mossa/partita durante la navigazione di un incontro),</li>
<li>il <strong>contatore della libreria</strong> — «412 posizioni · 38 blunder · 5 match» — dove ogni numero <strong>apre ciò che conta</strong>: le posizioni, la ricerca <code>E&gt;</code> alla soglia della libreria (avviata subito, come se fosse stata digitata, e registrata nella cronologia delle ricerche), o l'elenco dei match. Quando l'elenco a schermo non è l'intera libreria (ricerca, raccolta, match), la cifra dei blunder si scrive «12 / 340 blunder»: i blunder di quell'elenco, poi quelli della libreria, alla stessa soglia; un elenco di oltre 20 000 posizioni non viene contato e mantiene solo il totale. Una cifra che non si può seguire è una decorazione. La soglia dei blunder è quella della libreria, impostata nella scheda <em>Libreria</em> della configurazione e condivisa con le statistiche: due soglie farebbero dire due cose alla stessa parola. Il contatore promette esattamente ciò che il suo collegamento apre, anche per una posizione giocata in più modi, che vale il suo costo più elevato. In una libreria molto grande (oltre 200 000 righe) il contatore non scorre le tabelle: un numero preceduto da «≈» è una stima (un limite superiore, perché le righe eliminate lasciano dei vuoti). Se le posizioni sono stimate, il numero di blunder, che non ha una stima onesta, appare come «?» — il collegamento avvia la ricerca, che dà il conteggio esatto. I due numeri dei blunder sono link distinti: il primo avvia la sotto-ricerca <code>ss E&gt;</code> nella lista a schermo, il secondo la ricerca in tutta la biblioteca; quando compare solo il totale, c'è un solo link.</li>
</ul>
<div class="admonition note">
<p>Nel caso di posizioni derivanti da una ricerca dell'utente, il numero di posizioni indicato nella barra di stato corrisponde al numero di posizioni filtrate.</p>
</div>
<p>La scheda <strong>Anki</strong> porta un <strong>contrassegno</strong> quando ci sono carte da ripassare, in tutti i mazzi. Quella cifra è la ragione per aprire la scheda; non ha nulla da fare dietro di essa. Zero non mostra nulla: un contrassegno che dice «0» è rumore.</p>
<p>Il comando <code>log</code> apre il <strong>registro attività</strong>: le ultime duecento righe del file di log, un pulsante per copiarle — quanto serve per allegare un rapporto a una segnalazione — e un altro per aprire la cartella che le contiene. Il registro non è né filtrato né riformattato: un registro che si abbellisce è un registro che non si può più citare.</p>
<p>Il comando <code>grid</code> apre il <strong>provino</strong>: l'elenco sfogliato — risultati di una ricerca, libreria, collezione — come griglia di mini-tavolieri, disegnati come il tavoliere, in pagine da ventiquattro. Si apre sulla pagina della posizione corrente, la cui miniatura è incorniciata; un clic o <em>INVIO</em> su una miniatura ne apre la posizione sul tavoliere e chiude il provino, e si sfoglia interamente da tastiera (vedi Provino). Non si apre né in modalità modifica né in un match, che si sfoglia per mosse.</p>
<p>Nella <strong>cronologia delle ricerche</strong> del pannello Ricerca, ogni token di un comando salvato appare come un'etichetta con nome — <em>Senza contatto</em>, <em>Errore di mossa</em> — anziché come token nudo. Il comando esatto resta nel suggerimento, perché è quello che si rilancia; e un token che blunderDB non riconosce appare <strong>così com'è</strong>, non tradotto al più vicino.</p>
<h3>Schede delle viste</h3>
<p>Sotto la barra degli strumenti, una barra delle schede consente di lavorare con più <strong>viste</strong> in parallelo. Ogni vista è uno spazio di lavoro indipendente che conserva il proprio elenco di posizioni, l'indice della posizione corrente, la posizione visualizzata, l'analisi e la mossa selezionata, il pannello attivo, il commento in corso nonché il contesto di navigazione in un match. È così possibile, ad esempio, tenere aperta una ricerca in una vista mentre si scorre un match in un'altra.</p>
<ul>
<li><strong>Creare una vista</strong>: fare clic sul pulsante <em>+</em> della barra delle schede (chiamato <em>+ Nuova vista</em> finché c'è una sola vista) o premere <em>CTRL-T</em>. La nuova vista parte come copia della vista corrente.</li>
<li><strong>Chiudere una vista</strong> : fare clic sulla croce della scheda o premere <em>CTRL-W</em>. L'ultima vista non può essere chiusa.</li>
<li><strong>Cambiare vista</strong> : fare clic su una scheda, premere <em>CTRL-PageUp</em> / <em>CTRL-PageDown</em> (o <em>MAIUSC-J</em> / <em>MAIUSC-K</em>) per passare alla vista precedente / successiva, oppure da <em>CTRL-1</em> a <em>CTRL-9</em> per raggiungere direttamente l'n-esima vista.</li>
<li><strong>Rinominare una vista</strong> : fare doppio clic sulla scheda, inserire il nuovo nome e confermare con <em>INVIO</em>.</li>
</ul>
<p>Le viste vengono salvate con lo stato di sessione del database e ripristinate alla sua riapertura.</p>
<h3>Configurazione</h3>
<p>Il pulsante di configurazione (icona a forma di ingranaggio) situato nella barra degli strumenti, a sinistra del pulsante di aiuto, apre la finestra di configurazione di blunderDB. È organizzata in nove schede:</p>
<ul>
<li><strong>Interfaccia</strong> — tema, lingua, scala di visualizzazione, posizione del pannello, passo di PageUp / PageDown (10, 50, 100, 500 o 1 000 posizioni, oppure il 10 % dell'elenco), registri, verifica degli aggiornamenti, il vostro nome e posizioni vicine;</li>
<li><strong>Colori della board</strong> — i colori della board;</li>
<li><strong>Libreria</strong> — ciò che appartiene al database aperto: le soglie di errore e di blunder, la compattazione e la riparazione, descritte qui sotto;</li>
<li><strong>Corpus</strong> — i duplicati all'importazione, gli alias di giocatori ed eventi e la ricerca dei duplicati probabili (vedere l'importazione dei match);</li>
<li><strong>Bearoff</strong> — le tabelle di bearoff usate dal pannello Eval;</li>
<li><strong>gammonNet</strong> — le impostazioni del valutatore integrato, descritte qui sotto;</li>
<li><strong>Cartella sorvegliata</strong> — l'importazione automatica degli incontri che arrivano in una cartella, descritta più sotto;</li>
<li><strong>Assistente e MCP</strong> — il server MCP locale e l'assistente interno, descritti più avanti;</li>
<li><strong>Identità dell'emittente</strong> — la chiave che firma i tuoi contrassegni di origine, descritta nella sezione Distribuire un database: origine e password.</li>
</ul>
<p>La scheda <em>Interfaccia</em> comincia con un <strong>tema</strong>: <em>seguire il sistema</em>, <em>chiaro</em>, <em>scuro</em>, <em>contrasto elevato</em> o <em>stampabile</em>. Il tema regola i colori dell'interfaccia e <strong>propone una tavolozza per la dama</strong> — un'interfaccia scura attorno a una dama chiara non è un tema scuro, è la metà di uno, poiché la dama occupa gran parte della finestra.</p>
<p>Voi mantenete l'ultima parola, e il meccanismo lo garantisce anziché prometterlo: la scheda <em>Colori</em> continua a regolare la dama direttamente, e un colore scelto dopo il tema è il vostro. All'avvio sono applicati solo i token dell'interfaccia, mai la tavolozza della dama — quella che avete regolato è già caricata, e riscriverla a ogni lancio cancellerebbe il vostro lavoro una sessione alla volta. Vedere <code>ADR-0038 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0038-a-named-theme-carries-the-board-palette-and-the-user-still-has-the-last-word.md&gt;</code>__.</p>
<p><em>Seguire il sistema</em> è l'impostazione predefinita: obbedisce alla preferenza chiaro/scuro della scrivania, anche quando cambia a metà sessione. Uno strumento non impone il proprio chiaro o scuro a una scrivania che ha già deciso.</p>
<p>La scheda <em>Interfaccia</em> permette anche di scegliere la lingua fra inglese, francese, tedesco, italiano, spagnolo, finlandese, giapponese, greco e russo. Tutta l'interfaccia (barra degli strumenti, pannelli, messaggi, aiuto) è tradotta nella lingua selezionata. La scelta della lingua è salvata e conservata da una sessione all'altra.</p>
<p>La scheda <em>Libreria</em> riunisce ciò che appartiene al file aperto e non alla macchina. È vuota finché nessun database è aperto, e lo dice.</p>
<p>Porta anzitutto le due <strong>soglie</strong> che decidono il vocabolario dell'intera applicazione: una decisione è un <strong>errore</strong> non appena il suo costo raggiunge la soglia di errore, e questo errore è un <strong>blunder</strong> non appena raggiunge la soglia di blunder. Ogni blunder è un errore, quindi la prima soglia non può superare la seconda, e blunderDB rifiuta la coppia invertita. I valori si inseriscono in equità — «0,080» — l'unità di tutte le tabelle; la riga di comando, invece, parla in millipunti, sicché la soglia 0,080 si scrive <code>E&gt;80</code> in una ricerca.</p>
<p>Queste soglie seguono il file, non il computer: lo stesso database conta gli stessi blunder ovunque lo si apra, <code>blunderdb info</code> le mostra, e <code>blunderdb edit --error-threshold</code> / <code>--blunder-threshold</code> le imposta. Non viaggiano in un'esportazione: una soglia è un'abitudine di lettura, non un fatto delle posizioni.</p>
<p>Tre <strong>preimpostazioni</strong> sono proposte con un clic, ciascuna con il nome del programma che ha tracciato quella linea: blunderDB (0,050 / 0,100), XG (0,020 / 0,080) e gnubg (0,040 / 0,080). Per impostazione predefinita, una libreria legge 0,050 e 0,100.</p>
<p>Ciò che cambiano, sullo schermo: il numero di blunder del contatore della barra di stato e la ricerca che il suo collegamento prepara, le colonne «Errori» e «Blunder» delle statistiche e della tabella dei giocatori, i segni delle mosse nella scheda di un match (Pannello Match), e l'elenco delle posizioni che blunderDB propone di rivedere dopo un'importazione.</p>
<p>La stessa scheda offre anche il pulsante <strong>Compatta il database</strong>, che recupera lo spazio su disco lasciato dalle eliminazioni (match, tornei, purghe): il database non si riduce mai da solo quando si cancellano dati, questa compattazione va chiesta esplicitamente. L'operazione può richiedere tempo su un database grande. La copia compattata viene scritta accanto al file e poi lo sostituisce: occorre, temporaneamente, circa la dimensione del database in spazio libero su disco sul suo volume, e non in memoria. Se un altro programma ha il database aperto (per esempio una seconda istanza in sola lettura), il file non viene sostituito: il database è compattato sul posto, il che richiede circa il doppio della sua dimensione in spazio libero, e un file temporaneo va nella cartella temporanea del sistema, o in quella indicata dalla variabile d'ambiente <code>SQLITE_TMPDIR</code>: con un <code>/tmp</code> piccolo, puntatela al volume del database. blunderDB rifiuta di partire, con un messaggio che dice cosa manca (spazio su disco, o meno di 512 MB di memoria disponibile), anziché rischiare una compattazione interrotta. Prima di avviarla viene chiesta una conferma. Il risultato — lo spazio guadagnato, in megabyte — appare poi nella barra di stato. La stessa operazione è disponibile da riga di comando con <code>blunderdb vacuum</code> (vedere Interfaccia a riga di comando (CLI)).</p>
<p>Dopo una compattazione sul posto, blunderDB verifica che il file si sia davvero ridotto. Su Windows, il sistema rifiuta di accorciare un file che un altro programma tiene aperto in certi modi: la compattazione termina allora con un messaggio che lo dice — chiudete l'altro programma e rilanciatela — invece di annunciare un guadagno che non c'è stato.</p>
<p>Il pulsante <strong>Apri la cartella dei registri</strong>, nella scheda <em>Interfaccia</em>, apre la cartella che contiene il registro dell'applicazione — utile per allegare dettagli a una segnalazione di problema, in particolare quando blunderDB è stato avviato da un collegamento o da un doppio clic, senza un terminale collegato che mostri qualcosa.</p>
<p>La casella <strong>Verifica gli aggiornamenti all'avvio</strong>, della stessa scheda, disattivata per impostazione predefinita, interroga una volta a ogni avvio la pagina delle ultime versioni del repository GitHub e mostra nella barra di stato un messaggio se è disponibile una versione più recente — mai una finestra che blocchi l'uso. Questa verifica resta disattivata automaticamente su un'installazione passata da un gestore di pacchetti (Flatpak, Homebrew, un pacchetto di distribuzione…): è quel canale a gestire allora gli aggiornamenti, non blunderDB stesso.</p>
<p>Due impostazioni della scheda <em>Interfaccia</em> governano l'ordinamento delle <strong>posizioni vicine</strong> (token <code>like</code>, vedere Pannello Ricerca): il numero di vicine restituite e la distanza massima oltre la quale una posizione cessa di esserlo. Questa distanza vale zero per impostazione predefinita, cioè nessun limite: la scala dipende dalla fase della partita, e un valore scelto qui si leggerebbe come una misura. Il token <code>like&lt;12</code> impone la propria per una ricerca, senza toccare l'impostazione.</p>
<p>La scheda <em>Colori della board</em> permette di personalizzare i colori della board. Ogni elemento dispone di un proprio selettore di colore: lo sfondo, il bordo, le punte chiare e scure, le pedine del giocatore 1 e del giocatore 2, i dadi, i punti dei dadi e il cubo. Il pulsante <em>Reimposta</em> ripristina tutti i colori predefiniti. Come la lingua, i colori scelti vengono mantenuti da una sessione all'altra.</p>
<p>La scheda <em>Bearoff</em> gestisce le tabelle di bearoff del pannello Eval (vedi Pannello Eval). Non sono <strong>né incorporate nell'eseguibile né scaricate</strong>: blunderDB le calcola sulla macchina che le usa, e il risultato è identico byte per byte a ciò che produce gnubg — l'impronta SHA-256 è verificata prima che una tabella venga accettata.</p>
<p>Le due tabelle ordinarie (TS-06-06 per il verdetto di cubo, OS-06 per l'EPC) sono calcolate al primo avvio, in secondo piano e senza chiedere nulla: circa sei secondi su un core, durante i quali l'applicazione si usa normalmente. Il pannello Eval lo segnala solo se vi si pone una posizione che ha bisogno di una tabella non ancora pronta.</p>
<p>La scheda mostra il dominio attivo e la sua origine, lo stato della tabella a un lato che legge l'EPC, la cartella dove tutto questo vive, e l'elenco delle tabelle presenti con la loro dimensione e il loro verdetto. Ogni riga si elimina singolarmente, dopo conferma.</p>
<p><strong>Verificata o non verificata.</strong> Una tabella <em>verificata</em> ha esattamente i byte che gnubg produce per il suo dominio: la sua impronta SHA-256 figura in blunderDB ed è stata ritrovata. Le impronte registrate per le tabelle a un lato (da OS-06 a OS-10) sono quelle prodotte dallo strumento <code>makebearoff</code> di GNUbg 1.08. Una tabella <em>non verificata</em> è ben formata ma il suo dominio non ha impronta registrata — non le si rimprovera nulla, semplicemente nessuno l'ha confrontata con il riferimento. Una tabella <em>corrotta</em> si contraddice da sé e non viene mai letta; viene ricalcolata.</p>
<p><strong>Calcolare una tabella più ampia.</strong> Il dominio si sceglie in un elenco di due famiglie, con il numero di core da dedicarvi (per impostazione predefinita tutti tranne uno, perché la macchina resti utilizzabile):</p>
<ul>
<li><strong>cubo esatto (due lati)</strong>, da TS-06-06 a TS-06-15: amplia il dominio in cui la probabilità di vittoria e il verdetto del cubo sono letti anziché stimati;</li>
<li><strong>EPC fuori dalla casa (un lato)</strong>, da OS-06 a OS-10: amplia la distanza a cui una pedina può trovarsi senza che il blocco EPC taccia. Questo passaggio legge solo posizioni più piccole di quella che calcola, quindi è sequenziale per costruzione e il numero di core non gli serve — il selettore lo dice ingrigendosi.</li>
</ul>
<p>Prima di lanciare qualsiasi cosa, la scheda dichiara tre cifre per il dominio scelto: la dimensione su disco, la memoria necessaria durante il calcolo e il tempo che dovrebbe volerci <em>su questa macchina</em>. Quest'ultimo comincia come stima e diventa una misura: ogni calcolo abbastanza ampio rileva la propria velocità e la conserva. Un dominio che la memoria disponibile non permette è proposto in grigio, con la ragione — « servirebbero 24 GB, ne restano 12 » è una risposta, una riga assente non lo sarebbe.</p>
<p>Come ordine di grandezza, su una macchina a sedici thread: TS-06-09 pesa 191 MB e richiede una decina di secondi, TS-06-11 pesa 1,2 GB e qualche minuto, TS-06-13 supera ciò che la maggior parte delle macchine può tenere in memoria. Dal lato a un lato, su un core: OS-07 pesa 4,9 MB e richiede 17 s, OS-08 15 MB e 1 min 20, OS-10 117 MB e mezz'ora.</p>
<p><strong>Pausa e ripresa.</strong> Durante il calcolo, l'avanzamento mostra il tempo rimanente <em>misurato</em> e due pulsanti distinti: <em>Pausa</em> e <em>Annulla</em>. La pausa scrive lo stato del calcolo accanto alla tabella; rilanciarlo riprende da dove si era fermato invece di ricominciare. Annullare non conserva nulla. Chiudere la finestra di configurazione non interrompe niente — il calcolo prosegue in secondo piano.</p>
<p>Un calcolo messo in pausa si ritrova all'avvio successivo, nominato e quantificato («TS-06-09 interrotta al 43 %»), con <em>Riprendi</em> ed <em>Elimina</em>. Nulla riparte da solo: è l'utente ad aver chiesto l'arresto.</p>
<p>La scheda permette infine di puntare a un file <code>.bd</code> a due lati esterno, per esempio una base prodotta da gnubg stesso: vince la tabella dal dominio più ampio.</p>
<p>La scheda <em>Libreria</em> porta infine <strong>Riparare le analisi</strong>: le colonne di analisi che la ricerca e le statistiche interrogano sono una proiezione delle analisi memorizzate, le quali restano intatte. Un difetto di proiezione si ripara quindi senza reimportare nulla. È esplicito e mai automatico — riscrivere le colonne di analisi di qualcuno per il solo motivo che apre il proprio database non è una cosa che uno strumento debba fare alle sue spalle. Lo stesso <code>blunderdb repair</code> è disponibile nella riga di comando.</p>
<p>La scheda <strong>gammonNet</strong> regola il valutatore integrato (vedere <code>ADR-0011 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0011-gammonnet-is-ported-to-go-and-the-representation-boundary-sits-at-the-evaluator-s-edge.md&gt;</code>__). Vi si regolano due profondità di ricerca, denominate e conservate separatamente — abbassare l'una non modifica mai l'altra:</p>
<ul>
<li><strong>Profondità di visualizzazione</strong> — il comfort interattivo durante la modifica del tavoliere; mai scritta nel database.</li>
<li><strong>Profondità di analisi</strong> — ciò che il lotto di analisi dopo l'importazione scrive nell'Analisi di una posizione.</li>
</ul>
<p>Entrambe valgono per impostazione predefinita <strong>2-ply</strong>, la configurazione canonica. La scheda propone anche la <strong>potatura</strong> (predefinita <code>k=12</code>) e il <strong>numero di mosse candidate mostrate</strong> (predefinito 10), oltre a una casella <strong>analizza automaticamente dopo l'importazione</strong> che, una volta attivata, verifica dopo ogni importazione se restano posizioni <strong>senza alcuna analisi</strong> (né gammonNet, né XG, né GNUbg, né BGBlitz — la regola è « una valutazione colma soltanto una lacuna », mai una sostituzione) e, se è il caso, avvia in background un'analisi gammonNet alla profondità di analisi configurata. Un pulsante <strong>Analizza ora</strong> riavvia manualmente lo stesso recupero, utile per una libreria creata prima dell'esistenza di questa funzione.</p>
<p>Un secondo pulsante, <strong>Rianalizza le posizioni obsolete</strong>, copre il caso opposto: una posizione già analizzata da gammonNet, ma la cui analisi memorizzata è stata scritta con una versione del motore più vecchia di quella attualmente in esecuzione, o a una profondità diversa dalla profondità di analisi configurata sopra, viene lì segnalata come obsoleta e rivalutata. Una posizione che porta anche un'analisi XG, GNUbg o BGBlitz non viene mai toccata da questo pulsante, qualunque sia il suo contenuto gammonNet — la protezione di <code>ADR-0013 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0013-evaluations-fill-gaps-an-imported-analysis-is-never-overwritten.md&gt;</code>__ resta incondizionata. Il numero mostrato accanto a ciascun pulsante (posizioni senza analisi, posizioni obsolete) è puramente informativo; il lotto ricalcola il proprio elenco all'avvio.</p>
<p>Entrambi i lotti sono <strong>limitati, visibili e annullabili, mai un demone silenzioso</strong>: il loro avanzamento (<code>posizioni analizzate / totale</code>) e un pulsante di annullamento compaiono nella barra di stato per tutta la loro durata, e scompaiono una volta terminati a favore di un messaggio che riassume il risultato — quante posizioni sono state <strong>analizzate</strong>, quante sono state <strong>rifiutate</strong> (una posizione che gammonNet rifiuta di valutare, come un punteggio di partita fuori dalla portata della sua tabella, il che non è mai un guasto) e quante sono <strong>fallite</strong> (ritentate, invariate, alla prossima esecuzione). Chiudere l'applicazione durante l'uno o l'altro non perde nulla: ogni posizione analizzata viene scritta man mano, e la prossima esecuzione riprende esattamente da dove l'analisi si era fermata, senza alcun registro da tenere.</p>
<p>La stessa scheda contiene l'<strong>impostazione dei rollout</strong> avviati dal pannello Analisi (vedi Rollout), conservata da una sessione all'altra: <strong>Veloce</strong> (216 partite, troncate a 7 semimosse), <strong>Standard</strong> (1296 partite, troncate a 11 semimosse) o <strong>Libera</strong>, dove tutti i parametri sono modificabili — troncamento, partite minime e massime (multipli di 36), limite di JSD, profondità (ply), numero di candidate, seme e numero di processi.</p>
<p><strong>Tabella di equity del match della base.</strong> In fondo alla scheda, l'elenco <strong>Tabella di equity del match (MET) della base</strong> sceglie la tabella con cui gammonNet valuta i punteggi di match: Kazaross-XG2, integrata e predefinita, oppure una tabella importata da un file <code>.xml</code> di GNUbg con il pulsante <strong>Importa una MET .xml…</strong>. La scelta appartiene alla base, non all'applicazione: aprire un'altra base significa ritrovare la sua tabella. Sono lette solo le tabelle esplicite (non le tabelle parametriche <code>zadeh</code> o <code>mec</code>); oltre la lunghezza di una tabella importata, subentra la tabella integrata. Una tabella si identifica dai suoi valori, non dal suo nome: importare il <code>Kazaross-XG2.xml</code> di GNUbg non aggiunge nulla, è la tabella integrata. Ogni analisi calcolata da gammonNet registra la tabella con cui è stata calcolata; le analisi importate (XG, GNUbg, BGBlitz) sono considerate calcolate con Kazaross-XG2. Un'analisi a un punteggio di match calcolata con una tabella diversa da quella corrente è contrassegnata come <strong>MET diversa</strong> nel pannello di analisi ed esclusa dalle statistiche (medie degli errori, PR, classifiche, testa a testa); un'analisi in money game non lo è mai. Cambiare tabella non riscrive nessuna analisi: cambia solo ciò che i confronti considerano.</p>
<p>La valutazione in diretta del pannello e la matrice del cubo sono valorizzate con la tabella del database, come le analisi che registra. Una tabella viaggia con le analisi che la citano: l'esportazione di un database la porta con sé, e l'importazione di un database la aggiunge al database che la riceve, senza duplicati (una tabella già presente è riconosciuta dai suoi valori) e senza renderla corrente. Un'analisi gammonNet sostituita da quella di un altro motore (XG, GNUbg) torna a essere considerata calcolata con Kazaross-XG2.</p>
<p><strong>Un incontro importato senza analisi ottiene così un PR.</strong> È il caso di un incontro giocato online, o di un file Jellyfish <code>.mat</code>, che nessuno ha fatto passare da XG: blunderDB ne conosceva le posizioni e le mosse giocate, ma nessuna analisi diceva quanto valessero. Una volta passato il lotto, la mossa effettivamente giocata è confrontata con la classifica di gammonNet e lo scarto alimenta il PR, il tasso di errore, le peggiori decisioni e tutti gli altri indicatori, esattamente come per un incontro analizzato da XG. Il confronto non inventa nulla: la mossa giocata proviene dalla tabella delle mosse dell'incontro, scritta all'importazione, che il file portasse un'analisi o no.</p>
<p>Un database analizzato con una versione precedente a questa non ha bisogno di essere rivalutato: <code>blunderdb repair</code> ricalcola le colonne a partire dalle analisi e dalle mosse già in archivio e restituisce il loro PR a quegli incontri (vedere repair).</p>
<p>Una riserva onesta: una posizione è identificata dalla sua struttura, quindi una posizione incontrata due volte — giocata bene una volta, male l'altra — porta un solo scarto, quello della sua prima occorrenza registrata. Non è proprio di questo calcolo: una biblioteca XG ha esattamente la stessa forma.</p>
<h4>Cartella sorvegliata</h4>
<p>La scheda <strong>Cartella sorvegliata</strong> chiede a blunderDB di guardare una cartella mentre gira e di importare ogni file di incontro che vi <strong>compare</strong>. Giocare una sessione in eXtreme Gammon, tornare a blunderDB, e trovare gli incontri già lì.</p>
<p>Nulla è indovinato. Finché nessuna cartella è indicata non c'è sorveglianza: blunderDB non si mette a leggere una directory perché ha supposto dove vivono i vostri incontri. Il pulsante <strong>Proporre</strong> guarda i posti abituali su questa macchina e ne propone uno solo se esiste davvero; altrimenti lo dice, e indicare la cartella spetta a voi.</p>
<p>Tre punti meritano di essere noti prima di attivare la casella:</p>
<ul>
<li><strong>Sono importati solo i file che compaiono.</strong> Ciò che la cartella contiene già quando la sorveglianza parte è registrato come noto e lasciato in pace: puntare una sorveglianza su quattro anni di incontri non deve importarli tutti. Per importare ciò che c'è, usate l'importazione di cartella, che esiste per questo — e le due si compongono benissimo, prima l'importazione, poi la sorveglianza.</li>
<li><strong>Un file è importato solo quando la sua dimensione si è stabilizzata.</strong> Un incontro che un altro programma sta scrivendo cresce da un'occhiata all'altra; importarlo scritto a metà darebbe un errore di analisi su cui nessuno può agire. blunderDB attende quindi di vedere due volte lo stesso file immutato.</li>
<li><strong>L'importazione è silenziosa.</strong> Stavate studiando una posizione quando sono arrivate le vostre partite: togliervi lo schermo sarebbe il momento peggiore. La modalità, la ricerca attiva, la scheda e la posizione visualizzata non si spostano; l'elenco delle posizioni non viene ricaricato e mostra le nuove partite al prossimo ricaricamento. L'importazione avviene senza finestra, e la barra di stato mostra un banner con il conteggio delle partite importate, ignorate (duplicati) e fallite, con un pulsante che apre il rapporto completo se lo desiderate. Tutto il resto è identico a un'importazione manuale: gli stessi duplicati rilevati, lo stesso lotto di importazione, la stessa analisi automatica se è attivata.</li>
</ul>
<p>L'intervallo predefinito è di dieci secondi; il minimo è due. La cartella non è percorsa ricorsivamente: una cartella sorvegliata è il posto dove uno strumento deposita i suoi incontri, non un albero da esplorare. Una condivisione di rete smontata non ferma la sorveglianza e non fa nemmeno passare il suo contenuto per nuovo al ritorno.</p>
<p>La stessa sorveglianza esiste da riga di comando, con <code>blunderdb import --type batch --dir &lt;cartella&gt; --watch</code> (vedere Interfaccia a riga di comando (CLI)): è la forma che un server, un'attività pianificata o uno script possono usare.</p>
<h4>Assistente e MCP</h4>
<p>La scheda <strong>Assistente e MCP</strong> regola due cose, entrambe disattivate per impostazione predefinita.</p>
<p>Il <strong>server MCP locale</strong> offre gli strumenti di blunderDB (ricerca, lettura di una posizione e della sua analisi, statistiche di un giocatore, quiz…) a un assistente che parla il Model Context Protocol, come Claude Desktop o Claude Code, finché la finestra è aperta. Ascolta solo su questa macchina, all'indirizzo <code>http://127.0.0.1:&lt;porta&gt;/mcp</code> (porta 8765 per impostazione predefinita), e rifiuta una richiesta proveniente da una pagina web. Aggiunge due strumenti di visualizzazione: aprire una vista su una ricerca e mostrare una posizione. Gli strumenti si limitano a leggere, salvo se <strong>Consenti la scrittura</strong> è spuntata: possono allora salvare una posizione, commentarla, creare e riempire una collezione, valutare una carta Anki e registrare un rollout accanto all'analisi di una posizione; nulla cancella. Senza una finestra aperta, <code>blunderdb mcp</code> serve gli stessi strumenti (vedi Interfaccia a riga di comando (CLI)).</p>
<p>L'<strong>assistente interno</strong> è un client di questi stessi strumenti. Nessun modello è incorporato: si rivolge a un fornitore compatibile con OpenAI — Ollama su questa macchina per impostazione predefinita, oppure Groq, OpenRouter, Gemini, Anthropic o un altro indirizzo. È remoto ogni indirizzo esterno a questa macchina, qualunque sia il fornitore scelto: le vostre frasi e i risultati degli strumenti la lasciano allora, la scheda lo dice e attende il vostro consenso per quell'indirizzo preciso; un altro indirizzo lo richiede di nuovo. La chiave API va nel portachiavi del sistema, mai nel database né nel file delle impostazioni. Il modello proposto per impostazione predefinita per Ollama, <code>qwen2.5:7b</code>, è un punto di partenza, non una raccomandazione: nessuna raccomandazione di modelli viene fatta senza un punteggio del banco di misura fornito con il codice sorgente.</p>
<p>Una volta attivato, l'assistente è una sotto-scheda <strong>Assistente</strong> del pannello Ricerca. Una frase — «i miei errori di oltre 50 millipunti nella corsa» — apre una nuova vista col suo nome, vi avvia la ricerca e vi passa; i suoi token restano nella cronologia delle ricerche. Ciò che gli strumenti restituiscono proviene dal database; ciò che il modello scrive è contrassegnato come <strong>Testo libero del modello</strong>. L'assistente propone una modifica solo se la casella <strong>Lasciare che l'assistente proponga modifiche (ciascuna confermata)</strong> è spuntata — un'impostazione distinta dalla scrittura del server MCP locale — e ogni modifica che prepara viene mostrata ed eseguita solo dopo <strong>Conferma</strong>.</p>
<p>La finestra di configurazione raggruppa anche alcune impostazioni di visualizzazione dell'interfaccia. Un cursore di <strong>scala dell'interfaccia</strong> consente di ingrandire o ridurre l'insieme degli elementi, il che è utile sugli schermi ad alta densità o per migliorare la leggibilità. Un menu <strong>posizione dei pannelli</strong> determina la collocazione dei pannelli (ricerca, match, analisi) rispetto al tavoliere: <em>in basso</em>, <em>di lato</em> o <em>automatica</em> (il lato viene allora scelto sugli schermi larghi per sfruttare meglio lo spazio disponibile). Come le altre impostazioni, queste scelte vengono conservate da una sessione all'altra.</p>
<h3>Visite guidate e database di esempio</h3>
<p>Per facilitare i primi passi, blunderDB propone delle <strong>visite guidate</strong> dell'interfaccia. Il catalogo delle visite si apre dalla barra degli strumenti o con il comando <code>tutorial</code> (alias <code>tour</code>). Sono disponibili sette visite: una visita generale dell'interfaccia e visite dedicate alla ricerca di posizioni, alla revisione dei match, alla revisione dei tornei, al pannello Eval, al ripasso Anki e alle statistiche. Ogni visita evidenzia gli elementi interessati dell'interfaccia, passo dopo passo, apre strada facendo il pannello di cui parla, e può essere rigiocata in qualsiasi momento. Al primo avvio, la visita generale viene proposta automaticamente.</p>
<p>Il comando <code>demo</code> carica un <strong>database di esempio</strong> che permette di scoprire le funzionalità dello strumento senza importare le proprie partite: tre partite (due delle quali raggruppate in un torneo) analizzate da eXtreme Gammon, BGBlitz e gammonNet, tre raccolte tematiche, commenti con etichette (<code>#blunder</code>, <code>#cube</code>) e un mazzo Anki con il suo registro dei ripassi. Giocatori, torneo e luogo sono fittizi. Le visite guidate si basano su questo database quando nessun database è aperto.</p>
<h3>Navigazione tra le posizioni</h3>
<p>Per impostazione predefinita, blunderDB permette di:</p>
<ul>
<li>scorrere le diverse posizioni della libreria corrente — che non viene mai caricata in blocco: blunderDB ne tiene solo l'elenco degli identificatori e carica le posizioni per finestre di cinquanta attorno a quella visualizzata, cosicché un database di diverse decine di migliaia di posizioni si apre altrettanto velocemente di uno piccolo,</li>
<li>visualizzare le informazioni di analisi associate a una posizione,</li>
<li>visualizzare, aggiungere e modificare i commenti di una posizione.</li>
</ul>
<p>Il pulsante <strong>Vai alla posizione</strong> della barra degli strumenti apre una finestra in cui digitare direttamente l'indice di una posizione per saltarci, senza scorrere. È l'equivalente grafico del comando <code>[number]</code> della riga di comando (vedere Posizioni e navigazione).</p>
<div class="admonition tip">
<p>Fare riferimento a Scorciatoie da tastiera per le scorciatoie disponibili.</p>
</div>
<h3>Modifica delle posizioni</h3>
<p>La pressione del tasto <em>TAB</em> apre il pannello di ricerca e permette di modificare una posizione sul board per aggiungerla al database o per definire una struttura di posizione da cercare. La distribuzione delle pedine, il cubo, il punteggio e il turno possono essere modificati con il mouse (vedi Modificare una posizione).</p>
<div class="admonition tip">
<p>Fare riferimento a Scorciatoie da tastiera per le scorciatoie disponibili.</p>
</div>
<h3>La riga di comando</h3>
<p>La riga di comando, integrata nella barra di stato, permette di eseguire tutte le funzionalità di blunderDB disponibili nell'interfaccia grafica: operazioni generali sul database, navigazione delle posizioni, visualizzazione dell'analisi e/o dei commenti, ricerca di posizioni secondo filtri... Dopo aver preso confidenza con l'interfaccia, si raccomanda di utilizzare progressivamente la riga di comando, che consente un uso potente e fluido di blunderDB, in particolare per le funzionalità di ricerca delle posizioni.</p>
<p>Per aprire la riga di comando, premere il tasto <em>SPAZIO</em>. Per inviare una richiesta e chiudere la riga di comando, premere il tasto <em>INVIO</em>.</p>
<p>blunderDB esegue le richieste inviate dall'utente a condizione che siano valide e modifica immediatamente lo stato del database se necessario. Non sono richieste azioni di salvataggio esplicite da parte dell'utente.</p>
<div class="admonition tip">
<p>Fare riferimento a elenco dei comandi per l'elenco dei comandi disponibili nella riga di comando.</p>
</div>
<h3>La tavolozza dei comandi</h3>
<p>La tavolozza dei comandi (<em>CTRL-MAIUSC-P</em>) ritrova con un nome approssimativo ciò che non si sa più dove cercare: un comando della riga di comando, una scheda, un filtro della libreria o un match. Le lettere digitate devono comparire nell'ordine, non necessariamente attaccate, senza badare a maiuscole e accenti: «mtrcb» trova la matrice del cubo, «lyon» i match di un torneo di Lione.</p>
<p>Le frecce scelgono, <em>INVIO</em> esegue, <em>ESC</em> chiude. Un comando si esegue come se fosse stato digitato; <code>s</code> e <code>ss</code> aprono la riga di comando per scriverci i filtri; un filtro si esegue come con un doppio clic nella libreria; un match si apre come con un doppio clic nel pannello Match.</p>
<p>Quando una Direzione è aperta, la tavolozza vi aggiunge il torneo: giocatori, tavoli, match in corso e prove (vedi la ricerca rapida).</p>
<h3>Pannello Analisi</h3>
<p>Il pannello <strong>Analisi</strong> (<em>CTRL-L</em>) visualizza i dati di analisi della posizione corrente importati da eXtreme Gammon (XG), GNUbg, BGBlitz o gammonNet. Mostra le migliori alternative (mosse di pedine o decisioni di cubo) con i relativi valori di equity e gli errori corrispondenti. Il tasto <em>d</em> alterna tra l'analisi delle mosse di pedine e l'analisi del cubo. Durante la navigazione in un match, la mossa effettivamente giocata viene evidenziata nell'elenco delle alternative. Premere <em>CTRL-L</em> o eseguire il comando <code>list</code> per mostrare o nascondere il pannello.</p>
<p>Sotto le tabelle, una <strong>frase</strong> dice a volte quanto è costata la decisione giocata e perché: «Perdi 120 mp: la mossa giocata lascia tre pedine scoperte dove 13/7 8/7 ne lascia solo una.» Viene da sei regole misurabili — l'esposizione, un punto di casa fatto o mancato, le probabilità di gammon abbandonate, una sicurezza che costa più di quanto renda, e i due sensi di un errore di cubo (raddoppiare troppo tardi o troppo presto, prendere troppo largo o passare troppo stretto).</p>
<p>La regola che conta è quella del <strong>silenzio</strong>: la frase compare solo se una regola si applica con sicurezza, e su un errore oltre la soglia da cui i motori concordano che lo sia. Il resto del tempo non c'è frase — né cornice vuota, né «non lo sappiamo». Una spiegazione sbagliata costa più di nessuna spiegazione: insegna qualcosa di inesatto.</p>
<p>La stessa frase accompagna l'errore dove lo avete appena commesso: sul retro di una <strong>carta Anki</strong>, sotto l'analisi rivelata, e nel <strong>verdetto del quiz</strong> dell'esercizio Decisione, sotto il costo in mp. Valgono le stesse regole di silenzio: una mossa giusta, o un errore che nessuna regola spiega, non aggiunge nulla.</p>
<p>Quando una posizione è stata giudicata da <strong>più motori</strong>, una fascia in testa al pannello li mette fianco a fianco: una riga per motore, con la sua profondità e la sua risposta — il verdetto del cubo, o la sua propria mossa migliore. Dice anzitutto se concordano, ed è il disaccordo a giustificarla: «XG dice raddoppio, presa; gammonNet dice niente raddoppio» si legge a colpo d'occhio, là dove bisognava confrontare due tabelle in diagonale.</p>
<p>La mossa migliore di un motore è la migliore <strong>di quel motore</strong>: l'elenco delle mosse candidate è ordinato per equità, con tutti i motori mescolati, quindi il suo primo elemento non è la mossa migliore di nessuno in particolare.</p>
<p>La fascia appare solo se ci sono davvero più motori, ed esiste unicamente in questo pannello: il pannello Eval presenta <strong>una</strong> decisione, quella del motore integrato (<code>ADR-0017 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0017-the-panel-shows-position-facts-plus-the-one-decision-the-board-asks.md&gt;</code>__), e un confronto non vi avrebbe posto.</p>
<p>Le mosse sono scritte come si leggono sul tavoliere, qui come nel pannello Eval: la pedina meno avanzata si muove per prima, e <strong>una pedina che concatena più dadi si scrive una sola volta</strong> — un 64 giocato con la stessa pedina si legge <code>24/14</code>, e <code>24/14*</code> se colpisce all'arrivo. Il dettaglio della concatenazione ricompare solo quando dice qualcosa in più: un colpo <em>lungo il percorso</em> conserva il suo punto di passaggio, <code>24/18* 18/14</code>, senza il quale il colpo sul 18 sparirebbe dalla notazione.</p>
<p>L'equità di un'analisi importata segue la stessa regola del pannello Eval: la colonna indica il proprio referenziale, «Equity (money)» o «Equity (match)» a seconda del punteggio della posizione analizzata, mai un semplice «Equity» muto sulla scala. Le regole <strong>Jacoby</strong> e <strong>Beaver</strong> attive su una posizione money game vengono mostrate anch'esse, in badge sotto la tabella di decisione del cubo.</p>
<h4>Rollout</h4>
<p>Il pannello <strong>Analisi</strong> sa <strong>rollare</strong> una posizione: giocare centinaia di partite a partire da ogni mossa scelta, o da ogni azione del cubo, per distinguere due scelte che la valutazione diretta separa a malapena. Nella tabella delle mosse candidate, <em>Ctrl+clic</em> aggiunge o toglie una mossa dalla selezione e <em>Maiusc+clic</em> la estende fino alla mossa cliccata; il clic semplice seleziona una sola mossa, come d'abitudine. Un <strong>clic destro</strong> sulla selezione — o su un'altra mossa, che diventa la selezione — apre un menu la cui voce <strong>Rollout (Standard)</strong> avvia subito il rollout delle mosse selezionate con l'impostazione scelta. Su una decisione di cubo, il clic destro rolla l'intera decisione. Il tasto <em>r</em> del pannello fa lo stesso (senza selezione, rolla le migliori candidate della posizione), come il comando <code>rollout</code> (alias <code>ro</code>). Durante il rollout, una barra sottile sotto la tabella segue le partite giocate; <strong>Annulla</strong>, la voce <strong>Annulla il rollout</strong> del menu, <em>Esc</em> o ancora <em>r</em> lo fermano senza scrivere nulla. L'impostazione — <strong>Veloce</strong>, <strong>Standard</strong> o <strong>Libera</strong> — si sceglie nella scheda <strong>gammonNet</strong> della configurazione (vedi configurazione).</p>
<p>Lo stesso menu propone <strong>Copia la posizione e l'analisi</strong>, l'immagine che copia <em>CTRL-X CTRL-X</em>: la tavola e l'inizio dell'elenco delle mosse, oppure l'intera analisi del cubo. Non appena una mossa è selezionata, si aggiunge una seconda voce, <strong>Copia la posizione e le mosse selezionate</strong>: l'immagine conserva solo quelle mosse, nell'ordine della classifica, ciascuna preceduta dal suo rango fra tutti i candidati e con il suo scarto dalla migliore mossa della posizione, non dalla migliore della selezione. Su una decisione di cubo compare solo la prima. Le celle delle tabelle di analisi non si selezionano come testo: i clic di selezione non vi lasciano evidenziazioni.</p>
<p>Il risultato è <strong>memorizzato accanto all'analisi, mai al suo posto</strong>: un'analisi importata non viene modificata. Ogni mossa rollata porta il suo risultato nella propria riga della tabella, in una colonna <strong>Rollout</strong> che compare solo quando la posizione ha un rollout: l'equità e la semiampiezza del suo intervallo di confidenza al 95 %, sulla scala della colonna dell'equità. Passare il cursore sulla cella dà il dettaglio: la deviazione standard, la <strong>JSD</strong> (la distanza dalla mossa migliore in deviazioni standard della differenza: dal limite in poi, la mossa è decisa e non viene più giocata), il numero di partite e la <strong>Configurazione</strong> — il motore e la firma completa dei parametri: due rollout con la stessa firma sono gli stessi numeri. Il rollout si ferma non appena le mosse sono distinte. Il rollout di una decisione di cubo, che non ha una riga di mossa, viene mostrato come tabella sotto la decisione. Un rollout gioca il cubo nelle sue partite: la classifica è affidabile, l'equità assoluta un po' meno, come ricorda il blocco. Non gioca il beaver: la sua riga <em>raddoppio, presa</em> è quella di una semplice presa, anche su una posizione money con la regola Beaver, dove l'analisi diretta conta il beaver. Una posizione che non è nel database si può rollare, ma non viene memorizzata.</p>
<p>Il comando <code>ro search</code> rolla, una dopo l'altra, le posizioni della lista mostrata — risultati di ricerca, partita o collezione — che non portano ancora questo rollout; una conferma indica il totale prima di iniziare. Ogni posizione viene scritta appena finita: annullare conserva quanto è fatto, e rilanciare riprende da dove ci si era fermati. L'avanzamento sopravvive alla chiusura del pannello.</p>
<p>Un database aperto <strong>in sola lettura</strong>, perché un'altra istanza di blunderDB lo detiene, rifiuta subito un rollout che verrebbe memorizzato: un messaggio lo dice prima che sia giocata la prima partita.</p>
<h3>Pannello Commenti</h3>
<p>Il pannello <strong>Commenti</strong> (<em>CTRL-P</em>) mostra, aggiunge e modifica i commenti associati alla posizione corrente. Una posizione può averne diversi, scritti da persone diverse: sono tutti mostrati in un thread, dal più recente al più vecchio, ciascuno con il nome del suo autore, e i vostri vengono per primi. I commenti importati dai file XG sono associati automaticamente alle posizioni corrispondenti. Premere <em>CTRL-P</em> o eseguire il comando <code>comment</code> per mostrare o nascondere il pannello.</p>
<p>Il nome che firma i vostri commenti si imposta nelle preferenze, scheda <em>Interfaccia</em>, campo <strong>Il vostro nome</strong>; se è vuoto, i vostri commenti non sono firmati. I commenti di un'importazione XG sono firmati <code>XG</code>, quelli di un file di partita con il nome del suo trascrittore. Riscrivere un commento lo firma con il vostro nome. La ricerca <code>au"Alice"</code> trattiene le posizioni che Alice ha commentato; fuori dall'interfaccia, <code>blunderdb comment add --author</code> scrive un commento firmato e <code>blunderdb comment list</code> li rilegge (vedere Interfaccia a riga di comando (CLI)).</p>
<p>Ogni commento proveniente da un file porta un'<strong>etichetta di provenienza</strong> (<code>XG</code>, <code>GNU BG</code>, <code>BGF</code>, oppure <em>importato</em> quando la provenienza non è mai stata registrata). I commenti che hai scritto non ne portano: è il caso normale, e segnalarlo a ogni riga sarebbe rumore. Modificare un commento importato te lo attribuisce: dopo la modifica, la frase è la tua.</p>
<p>Questa distinzione si vede altrove: cancellare una partita non distrugge più una posizione su cui <strong>tu</strong> avevi scritto. Una nota ripresa dal file di origine, invece, sparisce con la partita che l'ha portata.</p>
<h4>I tag</h4>
<p>Un <strong>tag</strong> è una <code>#parola</code> scritta in un commento. Nulla lo dichiara, nessuna tabella lo porta, ed è voluto: il vocabolario è la vostra prosa, ed esigere una dichiarazione prima di poter taggare trasformerebbe un'abitudine in scartoffie.</p>
<p>Ciò che mancava era l'altra metà: <strong>vedere</strong> il vocabolario che ci si è costruito, e cliccare un tag invece di ricordare come lo si scriveva. Il comando <code>tags</code>, o il pulsante <code>#</code> accanto alla casella di scrittura, apre la finestra del vocabolario: i tag di questo database, ognuno con il <strong>numero di posizioni</strong> che lo portano, cliccabili per lanciare la ricerca corrispondente. Sotto l'elenco figurano i tag consigliati che il database non usa ancora — un vocabolario tratto dalla letteratura del backgammon (<code>#blitz</code>, <code>#prime</code>, <code>#holding</code>, <code>#backgame</code>, <code>#containment</code>, <code>#crunch</code>, <code>#ace-point</code>, <code>#timing</code>…), suggerito e mai imposto: un tag assente da quell'elenco vale esattamente quanto uno che vi figura.</p>
<p>Mentre si scrive, un <code>#</code> propone i tag che <strong>questo database</strong> usa già, poi quelli consigliati. È ciò che evita di scrivere <code>#back-game</code> un giorno e <code>#backgame</code> il giorno dopo, cosa che nient'altro coglierebbe.</p>
<p>La ricerca per tag si scrive <code>#prime</code> nella riga di comando. È <strong>delimitata</strong>: <code>#prime</code> non trova <code>#priming</code>, là dove una ricerca di testo ordinaria, che cerca una sottostringa, non sa distinguerli. Più tag si <strong>sommano</strong> — <code>s #prime #backgame</code> chiede le posizioni che portano entrambi — perché una posizione porta più tag: nominarne due non può che voler dire «entrambi». È l'opposto del filtro di fase o di provenienza, dove una posizione ha un solo valore e nominare due valori non può che voler dire «l'uno o l'altro».</p>
<p>La stessa lista si ottiene fuori dall'interfaccia con <code>blunderdb list --type tags</code> (vedere Interfaccia a riga di comando (CLI)).</p>
<h3>Il cestino</h3>
<p>Eliminare una posizione, una collezione, un commento o una carta Anki passa da un <strong>cestino</strong>: l'eliminazione avviene davvero, ma una copia di ciò che sparisce è conservata trenta giorni. Il comando <code>trash</code> apre la finestra che le elenca, ciascuna con <em>Ripristina</em> ed <em>Elimina</em>, e un pulsante <em>Svuota il cestino</em>.</p>
<p>Una posizione ripristinata torna con <strong>la sua analisi e i suoi commenti</strong> — restituirla nuda sarebbe un ripristino solo di nome. Non torna col suo vecchio numero: la riga d'origine non esiste più, e blunderDB la risalva tramite la sua impronta, il che garantisce che non crei mai un doppione ma le dà un nuovo identificativo. Una collezione torna con la sua lista; le posizioni che conteneva non erano mai state eliminate — una collezione è una vista su di esse.</p>
<p>Ciò che ha più di trenta giorni viene eliminato dal comando <code>vacuum</code>, mai dall'apertura di una base: non fare <code>vacuum</code> è tenere tutto.</p>
<div class="admonition note">
<p>Il cestino non viaggia. Un'esportazione non lo porta, ed eliminare una partita non ci mette nulla: la pulizia delle posizioni orfane che segue l'eliminazione di una partita è una manutenzione automatica, non un gesto dell'utente — vedi la regola di ritenzione in Pannello Match.</p>
</div>
<h3>Pannello Ricerca</h3>
<p>Il pannello <strong>Ricerca</strong> (<em>CTRL-F</em> o <em>TAB</em>) permette di filtrare le posizioni secondo criteri liberamente combinabili: struttura delle pedine, tipo di decisione di cubo, magnitudo dell'errore, date, tag, ecc. Il tasto <em>TAB</em> apre contemporaneamente il pannello di ricerca e l'editor di posizione, consentendo di definire una struttura di pedine da cercare direttamente sul board.</p>
<p>I filtri si impostano nella sottoscheda <strong>Criteri</strong>. Quando una ricerca non trova nulla, il pannello lo indica («Nessuna posizione corrisponde») con un pulsante <strong>Cancella i filtri</strong>, senza limitarsi alla barra di stato.</p>
<p>Per cercare tra le posizioni visualizzate, usare il comando <code>ss</code> seguito da filtri (es.: <code>ss nc</code>, <code>ss E&gt;40</code>). <code>ss</code> cerca nell'elenco sullo schermo: i risultati della ricerca precedente, la collezione aperta o le posizioni del match in revisione, sia che il comando venga digitato direttamente sia dal pannello di ricerca (<em>TAB</em>). La casella <em>Cerca nei risultati attuali</em> del pannello segue la stessa regola; la casella <em>Apri in una nuova scheda</em> mostra i risultati in una nuova vista (vedi Schede delle viste) anziché nella vista corrente. In una collezione e in un match, <code>s</code> è rifiutato: cercherebbe in tutta la libreria e sostituirebbe l'elenco visualizzato.</p>
<p>Dai risultati di una ricerca <code>ss</code> avviata da una collezione o da un match si esce con <em>Esc</em>, con una sola pressione non appena né un campo né il pannello che ha il focus ha qualcosa da chiudere (una mossa selezionata nell'analisi, per esempio): blunderDB torna alla collezione intera, o al match sulla mossa studiata, e alla posizione lasciata. Questo ritorno segue solo <code>ss</code>: <code>s</code>, avviato dal pannello di ricerca aperto su una collezione o un match, cerca in tutta la libreria, e <em>Esc</em> non riporta più all'elenco lasciato.</p>
<p>Una ricerca su un database grande viene mostrata prima di essere contata: la prima pagina di risultati compare subito, e la barra di stato indica « Ricerca… » con il tempo trascorso finché il totale non è noto; questo sostituisce allora la lunghezza provvisoria dell'elenco. Una sola ricerca alla volta è in corso: avviarne un'altra abbandona la precedente. <em>Esc</em> interrompe la ricerca in corso e ferma la sua scansione del database; se la prima pagina era già visualizzata, resta da sola, e la barra di stato lo dice.</p>
<p>Il pannello offre un controllo esplicito del <strong>tipo di decisione</strong> ricercato: <em>Indifferente</em> (nessun filtro), <em>Pedine</em> (decisioni di mossa) o <em>Cubo</em> (decisioni di cubo). Quando è selezionato <em>Cubo</em>, un secondo elenco precisa il sotto-tipo: <em>Tutti</em>, <em>Raddoppio / No raddoppio</em> (il giocatore di turno deve decidere se raddoppiare) o <em>Accetta / Passa</em> (risposta a un raddoppio avversario). Il controllo è sincronizzato con il board: modificare i dadi o il cubo sul board aggiorna il tipo di decisione, e viceversa. In modalità <em>Accetta / Passa</em>, il cubo è mostrato al centro del board al valore offerto; tale valore resta modificabile.</p>
<p>La <strong>fase di gioco</strong> — apertura, mediogioco, corsa, uscita delle pedine — è un'etichetta che blunderDB calcola dalla sola tavola. Non è mai modificabile ed è cercabile tramite il token <code>ph:</code> della riga di comando (<code>ph:race</code>, ripetibile: <code>ph:race ph:bearoff</code>). Tre delle sue quattro frontiere sono quelle che GNU Backgammon usa per indirizzare le sue reti; la quarta, dove finisce l'apertura, è una convenzione di blunderDB: una posizione è ancora in apertura finché nessuno dei due campi ha mosso più di quattro pedine dai propri punti di partenza, nessuna è uscita e nessuna è sulla barra.</p>
<div class="admonition note">
<p>L'etichetta viene ricalcolata dal comando <code>blunderdb repair</code>. Su una base aperta per la prima volta con questa versione, il calcolo avviene una volta, all'apertura. Una base le cui fasi non sono mai state calcolate non restituisce nulla per <code>ph:</code> — nulla, piuttosto che una risposta sbagliata.</p>
</div>
<p>Il token <code>like</code> <strong>ordina</strong> invece di restringere: la sua presenza dispone il risultato per distanza crescente da una posizione bersaglio — <code>like</code> quella corrente, <code>like42</code> quella di indice 42 — e gli altri token restringono l'insieme così ordinato, tanto che <code>s like42 E&gt;80</code> si legge «le vicine della 42 dove ho sbagliato». La distanza è una distanza di trasporto in pip di pedina, la quantità di movimento di pedine che separa due posizioni, vista dal giocatore di turno.</p>
<p>Una vicina è lo stesso <strong>problema</strong>, non lo stesso disegno: l'ordinamento si prende dentro la classe del bersaglio — stesso tipo di decisione, stesso regime (soldi o incontro) per una decisione di cubo, e un incontro diverso dal suo, perché le posizioni che la circondano nella sua stessa partita sono le sue strutture più vicine senza esserne mai le vicine. Dadi, punteggio e cubo restano fuori classe; i token ordinari li filtrano quando lo si vuole. <code>like42*</code> allarga la classe a tutti i tipi di decisione e a entrambi i regimi, mai all'incontro del bersaglio; <code>like&lt;12</code> scarta ciò che dista più di dodici pip di pedina. Un ordinamento che non trova nulla rende una lista vuota e lo dice, invece di dieci posizioni senza rapporto.</p>
<p>In modo <strong>modifica</strong>, <code>s like</code> prende per bersaglio la dama <strong>disegnata</strong>: si disegna all'incirca la posizione di cui ci si ricorda, si lancia, e la biblioteca risponde — là dove la ricerca per struttura esige il disegno esatto. La dama è allora letta come una posizione e non come un motivo: un punto lasciato vuoto conta come pedine fuori, il che è esatto per una posizione reale e falsa il calcolo per un disegno lasciato a metà.</p>
<p>Ogni vicina porta la sua distanza sotto le tabelle di analisi, insieme alla posizione di cui è vicina. È questo che permette di giudicare se si guarda una vicina o una coincidenza, ed è la ragion d'essere del limite. L'ordinamento si lancia anche senza passare dalla riga di comando: <em>CTRL-MAIUSC-L</em>, oppure la voce <strong>Posizioni vicine</strong> del menu contestuale della dama.</p>
<p>Il token <code>n</code> conta gli <strong>incontri</strong>: <code>n&gt;3</code> mantiene le posizioni incontrate almeno tre volte nella base, tutte le partite e tutti i giocatori insieme. È una domanda diversa da «cosa ho sbagliato» — una posizione incontrata venti volte e ben giocata diciannove resta quella da sapere a memoria. Si contano decisioni, non partite: la stessa posizione due volte in una partita vale per due, perché erano due decisioni. Combinato con un filtro giocatore, conta solo le occorrenze di quel giocatore: <code>n&gt;3 pl!"Alice"</code> mantiene le posizioni che Alice ha dovuto giocare almeno tre volte, e <code>pl"Alice"</code> quelle delle partite che ha disputato; <code>op"Bob"</code> limita allo stesso modo il conteggio alle partite contro Bob.</p>
<p>Il <strong>piano di gioco</strong> è una seconda etichetta derivata, accanto alla fase, e risponde alla domanda che un pacchetto di filtri salvati non sa porre: «mostrami i miei errori in holding game». Token <code>gt:</code>, ripetibile (<code>gt:holding gt:mutualholding</code>), dal punto di vista del <strong>giocatore di turno</strong> — il piano in cui la decisione veniva presa.</p>
<p>I dieci piani riconosciuti, nell'ordine in cui le regole li esauriscono, dal più specifico al più generale:</p>
<ul>
<li><code>race</code> — le pedine più arretrate dei due schieramenti si sono incrociate: nessun contatto è più possibile. Frontiera di GNU Backgammon.</li>
<li><code>bearin</code> — il giocatore di turno rientra le pedine mentre l'avversario tiene ancora un'ancora nella sua casa.</li>
<li><code>crunch</code> — il giocatore di turno ha al più sei pedine fuori dai suoi punti 1 e 2. Regola di GNU Backgammon, soglia del suo autore.</li>
<li><code>backgame</code> — due o più ancore nella casa avversaria.</li>
<li><code>acepoint</code> — una sola ancora, sul punto uno avversario, con almeno venti pip di ritardo.</li>
<li><code>blitz</code> — tre o più punti di casa fatti, e l'avversario alla barra o con una pedina scoperta da colpire in quella casa.</li>
<li><code>primevprime</code> — entrambi tengono un blocco di almeno quattro punti, e ciascuno ha una pedina intrappolata dietro quello dell'altro.</li>
<li><code>mutualholding</code> — entrambi tengono un'ancora alta.</li>
<li><code>holding</code> — il giocatore di turno tiene un'ancora alta, l'avversario no.</li>
<li><code>contact</code> — contatto, e nessuno dei piani sopra. L'apertura finisce qui.</li>
</ul>
<p>Tre di queste regole sono quelle di GNU Backgammon e sono documentate; le altre sono <strong>convenzioni di blunderDB</strong>. La letteratura del backgammon descrive i piani di gioco senza quantificarne le frontiere, e nessuna misura di accordo tra classificatori è pubblicata per questo problema. Le soglie non documentate — tre punti di casa per un blitz, quattro punti per un blocco, venti pip di ritardo per un ace-point game — sono quindi enunciate qui invece di restare nascoste nel codice, e sono versionate: cambiarle ed eseguire <code>blunderdb repair</code> rietichetta l'intera base.</p>
<div class="admonition note">
<p>Una sola etichetta è conservata per posizione, quella del giocatore di turno. Un'etichetta derivata non è mai modificabile, mai esportata come verità, e una base i cui piani non sono mai stati calcolati non restituisce nulla per <code>gt:</code> — come per <code>ph:</code>.</p>
</div>
<p>Il filtro <strong>Contrassegnata</strong> conserva le posizioni che avete contrassegnato nel software di origine della partita. Solo eXtreme Gammon produce questa informazione, registrata mossa per mossa nel file <code>.xg</code>; blunderDB la legge all'importazione e la conserva. Una decisione di cubo contrassegnata dà due posizioni contrassegnate, il raddoppio e l'accetta/passa, poiché blunderDB divide in due ciò che il file di origine registra come una sola decisione.</p>
<div class="admonition note">
<p>La marcatura non è retroattiva: le partite già presenti nel database non contengono questa informazione, poiché esiste solo nei file di origine. È sufficiente reimportare il file <code>.xg</code> interessato — l'importazione rileva il duplicato e non aggiunge altro che i contrassegni, senza toccare i commenti né le analisi esistenti. Il contrassegno non può essere né posto né rimosso da blunderDB: per un elenco di lavoro temporaneo, utilizzate piuttosto una collezione.</p>
</div>
<p>Il filtro <strong>Commento</strong> interroga i commenti associati alle posizioni secondo tre modalità esclusive. <em>contiene il testo</em> cerca una o più parole nel testo dei commenti (campo di immissione, parole separate da <code>;</code>, almeno una deve corrispondere); <em>ha un commento</em> conserva ogni posizione che porti un commento, qualunque sia il suo contenuto; <em>senza commento</em> conserva al contrario le posizioni non annotate — utile, combinato con un filtro di errore o di data, per stilare l'elenco di ciò che resta da commentare.</p>
<div class="admonition note">
<p>I commenti importati da un file di partita (XG, GNUbg) contano come commenti. Per tenere solo i tuoi, aggiungi il token <code>co:user</code> sulla riga di comando (<code>co:xg</code>, <code>co:gnubg</code>, <code>co:bgf</code> e <code>co:unknown</code> designano le altre provenienze). Del resto, i commenti associati a una <em>partita</em> o a un <em>torneo</em> non sono interessati: annotano la partita o il torneo, non le sue posizioni.</p>
</div>
<p>Il filtro <strong>Match &amp; Tornei</strong> si basa su un selettore comune (finestra modale) anziché sull'inserimento di identificativi numerici: due elenchi con caselle di spunta, uno per i match e uno per i tornei, ciascuno filtrabile per testo (giocatore, data, evento per i match; nome, data, luogo per i tornei), con pulsanti <em>Tutti</em> / <em>Nessuno</em> che agiscono solo sul sottoinsieme attualmente filtrato. Selezionare un torneo seleziona automaticamente (e disabilita, mostrandoli in grigio) i match che ne fanno parte nell'elenco dei match, rendendo visibile il fatto che un torneo equivale all'insieme dei suoi match.</p>
<p>Il pannello di ricerca presenta tre schede sul bordo sinistro: <em>Criteri</em> (i filtri), <em>Cronologia</em> e <em>Salvati</em> — e una quarta, <em>Assistente</em>, quando l'assistente interno è attivato. La scheda <strong>Cronologia</strong> elenca le ricerche passate con la loro data e il loro comando: un clic seleziona una ricerca e mostra la posizione associata sul tavoliere, un doppio clic la riesegue. Ogni voce può essere salvata nella libreria dei filtri (icona segnalibro, dando un nome al filtro) o eliminata. La scheda <strong>Salvati</strong> contiene la <strong>libreria dei filtri</strong>: fare doppio clic su un filtro salvato per rilanciare la ricerca corrispondente (vedi Appendice: Utilizzo avanzato dei filtri). Il comando <code>history</code> (alias <code>hi</code>) apre il pannello di ricerca.</p>
<p>La stella di un filtro della libreria lo <strong>fissa</strong>. I filtri fissati compaiono come etichette in cima al pannello, qualunque scheda sia aperta, numerati nell'ordine della libreria: un clic su un'etichetta lancia il filtro, e <code>ALT-1</code> … <code>ALT-9</code> lanciano il filtro fissato di quel rango da qualsiasi schermata in modalità NORMAL o EDIT, senza aprire il pannello. Il filtro pone allora la stessa domanda del doppio clic, struttura delle pedine compresa. La fissazione appartiene al database: segue il filtro rinominato, scompare con il filtro eliminato e non viaggia con l'esportazione della libreria.</p>
<p>Una ricerca rilanciata conserva la sua classifica: <code>s like42</code> classifica rispetto alla posizione 42, e <code>s like</code> rispetto alla tavola salvata con la ricerca — quella che si stava consultando o disegnando. Una voce che non ha conservato la tavola non viene rilanciata contro quella sullo schermo, e la barra di stato lo segnala.</p>
<div class="admonition tip">
<p>Fare riferimento a elenco dei comandi per l'elenco dei filtri disponibili.</p>
</div>
<h3>Pannello Raccolte</h3>
<p>Nei pannelli Collezioni, Tornei, Anki e Trascrizione, il pulsante <strong>+</strong> dell'intestazione, seguito dal nome di ciò che crea (<strong>+ Nuova collezione</strong>, <strong>+ Nuovo torneo</strong>, <strong>+ Nuovo mazzo</strong>, <strong>+ Nuova trascrizione</strong>), è l'unico gesto di creazione: apre il campo di inserimento, che <em>Esc</em> o <strong>Annulla</strong> chiude nei pannelli Collezioni e Tornei. Nell'elenco dei match, l'icona ⌨ apre la trascrizione del match e l'icona ✎ ne corregge i metadati.</p>
<p>Il pannello <strong>Collezioni</strong> (<em>CTRL-B</em>) consente di gestire collezioni di posizioni. Le collezioni possono essere create, rinominate ed eliminate. Vi si possono aggiungere o togliere posizioni (tasto <em>Canc</em>, viene chiesta conferma). Fare doppio clic su una collezione per scorrerne le posizioni con i tasti <em>SINISTRA</em> e <em>DESTRA</em>. Il comando <code>ss</code> cerca tra le posizioni della collezione aperta; <em>Esc</em> riporta poi alla collezione (vedere Pannello Ricerca). L'ordine delle collezioni e delle posizioni all'interno di una collezione può essere modificato per trascinamento. Premere <em>CTRL-B</em> o eseguire il comando <code>collection</code> per mostrare o nascondere il pannello.</p>
<p>La <strong>Pila</strong> è la raccolta del gesto «rivedere più tardi»: un doppio clic fuori dalla scacchiera, il tasto <em>b</em> o il pulsante segnalibro della barra degli strumenti mette la posizione mostrata sulla Pila, e lo stesso gesto la toglie. Un segnalibro all'angolo della scacchiera indica che la posizione vi si trova, senza altro messaggio. Il gesto vale ovunque sia mostrata una posizione: revisione, ricerca, modifica, Eval, Trascrizione, quiz, Duello. L'azzeramento della scacchiera passa per il menu del clic destro (vedi menu della scacchiera). La Pila è una raccolta ordinaria — rinominata, riordinata, esportata, svuotata come le altre; è creata al primo uso e ricreata se è stata eliminata. Una posizione che non è ancora nella libreria (una bozza della scacchiera di ricerca o di valutazione) vi è scritta all'istante, come una posizione portata da sola, poi messa sulla Pila. La riga di comando fa lo stesso gesto con <code>collection pile</code>.</p>
<p>Una raccolta può essere <strong>viva</strong>: il suo contenuto non è più una lista fatta a mano ma il risultato di una <strong>ricerca</strong>, rivalutato ogni volta che la si apre. Il pulsante ◇ in testa alla raccolta la rende viva con l'ultima ricerca lanciata; ◈ segnala che lo è già, e lo stesso pulsante le restituisce la lista. Nulla viene distrutto: le posizioni che conteneva sono ancora lì quando si torna indietro.</p>
<p>Il pulsante ❄, visibile su una raccolta viva, la <strong>congela</strong>: le posizioni che la ricerca seleziona in quel momento diventano il contenuto di una raccolta ordinaria, nell'ordine della ricerca, e la query viene cancellata. Le posizioni che conteneva prima di essere viva vengono sostituite.</p>
<p>Una raccolta viva la cui interrogazione porta un token che questa versione non conosce più <strong>rifiuta di aprirsi</strong> e lo dice, invece di restituire l'intera base. È l'unico guasto che un filtro salvato non deve avere: allargarsi in silenzio.</p>
<h4>Lezioni</h4>
<p>Una <strong>lezione</strong> è una sequenza di passi che un coach scrive una sola volta per un allievo e gli consegna in un file di database (vedi il comando <code>lesson export</code> in cli). Ogni passo ha un titolo, un testo e può mostrare una collezione, una posizione, entrambe o nessuna. Il comando <code>le</code> elenca le lezioni del database nella barra di stato; <code>le 2</code> apre la lezione 2; <code>le edit</code> apre l'editor delle lezioni.</p>
<p>Compare allora una <strong>barra di lettura</strong> sopra la scacchiera: nome della lezione, numero del passo, titolo, poi il testo. <em>Precedente</em> e <em>Successivo</em> cambiano passo; il passo porta sulla scacchiera la collezione o la posizione che mostra, che si scorre poi con i gesti consueti. <em>Chiudi</em> esce dalla lezione. Un passo la cui collezione o posizione è stata eliminata conserva il suo testo.</p>
<p>La casella <em>Passo fatto</em> della barra segna il passo corrente come fatto; un secondo clic toglie il segno, e la barra conta i passi fatti. È l'unico gesto che scrive qualcosa: leggere una lezione, cambiare passo o aprirla non registra nulla, né il passo raggiunto né l'apertura. Il segno viene scritto nel database aperto, quello dell'allievo, e nessuna esportazione lo porta con sé. Importare un file che contiene una lezione la crea; una lezione con lo stesso nome già presente non viene toccata.</p>
<p><strong>L'editor delle lezioni</strong> si apre con <code>le edit</code> (<code>le edit 2</code> sulla lezione 2) o con il pulsante <em>Modifica</em> della barra di lettura. A sinistra, l'elenco delle lezioni e un campo per crearne una; a destra, il nome e la descrizione della lezione scelta, poi i suoi passi. Ogni passo ha un titolo, un testo, una collezione scelta dall'elenco e una posizione: <em>Posizione corrente</em> vi allega la posizione mostrata sulla scacchiera, <em>Stacca la posizione</em> la toglie. <em>Salva il passo</em> scrive le sue modifiche; le frecce lo spostano di un posto; <em>Aggiungi un passo</em> ne aggiunge uno in fondo. <em>Leggi</em> chiude l'editor e apre la lezione al suo primo passo; <em>Elimina</em> cancella la lezione e i suoi passi, senza toccare le collezioni né le posizioni che mostravano. Le lezioni si creano e si modificano anche da riga di comando o tramite l'API (Le lezioni).</p>
<p>Un backup — l'esportazione dell'intera libreria dalla finestra di esportazione o dalla riga di comando — include tutte le lezioni. Nella finestra di esportazione, la casella <em>Includi le lezioni</em> le sceglie una per una: ogni lezione spuntata parte con le collezioni e le posizioni che i suoi passi mostrano, e il file può portare un contrassegno di origine o essere protetto da password (<code>.dbx</code>) come ogni esportazione. Senza questa casella, un'esportazione parziale (una selezione di posizioni, collezioni o partite) non include le lezioni.</p>
<h3>Importazione: cosa viene scritto, cosa non lo è mai</h3>
<p>Importare un match, una posizione o un altro database aggiunge ciò che manca; non sostituisce ciò che è già presente.</p>
<ul>
<li><strong>Una posizione non è mai duplicata.</strong> È la sua identità — pedine, cubo, dadi, punteggio — a riconoscerla, mai il file da cui proviene: la stessa posizione incontrata in due match resta un'unica riga.</li>
<li><strong>Un'analisi per motore.</strong> eXtreme Gammon, GNUbg, BGBlitz e il valutatore integrato convivono sulla stessa posizione, e il pannello Analisi indica l'origine di ciascuna. Importarne una non cancella l'altra.</li>
<li><strong>Un'analisi importata non viene mai ricalcolata.</strong> blunderDB la conserva così com'è, con la sua etichetta di livello (« 3-ply », « XG Roller++ », « Book »), le sue equità, i suoi errori, le sue probabilità e la fortuna del lancio. La regola è « una valutazione colma soltanto una lacuna »: l'analisi automatica dopo l'importazione visita solo le posizioni senza <strong>alcuna</strong> analisi, e <em>Rianalizza le posizioni obsolete</em> lascia intatta ogni posizione che porta un'analisi importata (vedere Configurazione).</li>
<li><strong>Reimportare un match già presente porta solo ciò che è più profondo.</strong> Il match è riconosciuto dal suo gioco (giocatori, lunghezza, dadi, mosse, cubo), non dalla sua analisi: nessun match, partita o mossa viene riscritto. Le marcature poste nel software d'origine vengono aggiunte, e un'analisi più profonda di quella salvata la sostituisce, posizione per posizione — una versione Roller++ dello stesso match sostituisce la versione 3-ply, qualunque sia l'ordine d'importazione; a profondità uguale o minore, l'analisi salvata resta. Reimportare lo stesso file quindi non riscrive nulla. Il rapporto d'importazione distingue i duplicati che non portavano nulla da quelli che hanno approfondito delle analisi. Da riga di comando, <code>--skip-duplicates</code> salta un duplicato senza riprenderne altro che le marcature. Un match troncato e poi completato (più partite) non è lo stesso match: viene importato come un secondo match.</li>
<li><strong>Un'analisi illeggibile non impedisce l'importazione del match.</strong> Una decisione la cui analisi contiene un valore che non è un numero finito (NaN o infinito, presenti in alcuni file XG) viene importata senza quell'analisi; il resto del match entra normalmente, e il resoconto dell'importazione conta le decisioni interessate.</li>
<li><strong>Un match già presente con altri nomi viene segnalato, mai unito.</strong> Le due impronte del match includono i nomi dei giocatori: «Martin A.» e «Alice Martin» fanno due match. L'importazione confronta anche i dadi (lunghezza, punteggio iniziale, dadi di ogni partita) e segnala, sotto la riga del file e nel rapporto, il match già presente nel database con altri nomi di cui ha i dadi. <code>blunderdb repair --duplicates</code> elenca queste coppie in un database esistente, insieme ai match troncati e alla loro versione più lunga.</li>
<li><strong>Un nome noto con un'altra grafia viene registrato con il suo nome canonico.</strong> Un <em>alias</em> dice che «Martin A.» è un'altra grafia di «Alice Martin» (o che un nome di evento è un'altra grafia di un altro). All'importazione, i nomi dei giocatori e dell'evento vengono sostituiti dal loro nome canonico, e il match viene collocato nel torneo dell'evento canonico. Le impronte del match conservano i nomi del file: un file importato prima che il suo alias esistesse si riconosce quindi sempre. Un match i cui dadi sono quelli di un match presente nel database, e i cui nomi differiscono solo per alias noti, non è un secondo match: le sue analisi arricchiscono quello archiviato.</li>
</ul>
<p>Gli alias si gestiscono nella scheda <strong>Corpus</strong> delle impostazioni: scegliere <strong>Giocatori</strong> o <strong>Eventi</strong>, inserire l'alias e il nome canonico, oppure togliere un alias dall'elenco. <strong>Proponi</strong> elenca i nomi che differiscono solo per maiuscole, accenti, punteggiatura o ordine delle parole; non viene applicato nulla senza un clic su <strong>Applica</strong>. La stessa scheda contiene la casella <strong>Ignora i duplicati all'importazione</strong> (l'equivalente di <code>--skip-duplicates</code>, per la sessione) e la ricerca dei <strong>duplicati probabili</strong> di un database esistente, quella di <code>blunderdb repair --duplicates</code>: ogni coppia è data dai suoi numeri di match, nulla viene unito. Sotto una coppia con gli stessi dadi, <strong>Crea questi alias</strong> registra con un clic gli alias di giocatore che farebbero nominare ai due match gli stessi giocatori, la grafia del match più recente diventando l'alias: una sola proposta quando un nome è comune a entrambi, altrimenti due — posto per posto, poi incrociata — tra cui si sceglie quella che dice chi è chi. Da riga di comando: <code>blunderdb players alias</code> e <code>blunderdb events alias</code>.</p>
<ul>
<li><strong>Una cartella si importa in parallelo.</strong> I file vengono letti su più core contemporaneamente e scritti a gruppi, sempre nell'ordine della cartella: i numeri dei match non dipendono dalla macchina. Un file identico, byte per byte, a un file già letto della stessa cartella non viene riletto: conta come duplicato. Annullare ferma l'importazione al gruppo in corso; ciò che era già scritto resta.</li>
<li><strong>L'avanzamento si legge in posizioni al secondo.</strong> La finestra di importazione mostra la percentuale letta, la velocità, il tempo rimanente stimato e i conteggi in corso (importati, duplicati, in errore). <strong>Riduci</strong> la sposta nella barra di stato, da cui un'etichetta la riapre: l'importazione continua mentre lavori, e la finestra torna da sola con il rapporto alla fine. I primi cento errori sono elencati; i successivi sono contati, e il registro dell'applicazione li nomina tutti. Dalla riga di comando, <code>blunderdb import --type batch</code> mostra lo stesso avanzamento sullo standard error, e <code>--format json</code> lo restituisce nell'oggetto finale (<code>progress</code>).</li>
<li><strong>Ogni file di un'importazione viene registrato, e un'importazione interrotta può essere ripresa.</strong> Per ogni file, il registro del lotto conserva il percorso, la dimensione, la data di modifica, l'impronta SHA-256 e il risultato: partita nuova (con il suo numero), duplicato (con la partita che lo copre), partita arricchita o errore (con il messaggio). Un'importazione annullata o interrotta si continua senza ripetere ciò che è già deciso: un file con lo stesso percorso, la stessa dimensione e la stessa data viene saltato senza essere letto; un file con lo stesso contenuto viene letto ma non analizzato; un file in errore viene riprovato. Nell'applicazione, la finestra di fine importazione elenca il registro (un file per riga, con il suo risultato e il messaggio di un errore; un file riprovato mostra il suo ultimo esito), e una riga che ha prodotto una partita la apre. Un'importazione <strong>annullata</strong> lascia la finestra aperta con il pulsante <strong>Riprendi</strong>. Da riga di comando, <code>blunderdb import --type batch --dir &lt;cartella&gt; --resume &lt;lotto&gt;</code> riprende il lotto il cui numero è stato mostrato all'avvio dell'importazione, e <code>--format json</code> restituisce il registro nell'oggetto finale (<code>journal</code>). Il server accetta <code>resume</code> nella richiesta di <code>imports.batch</code> e restituisce il registro tramite <code>imports.files</code>. Il registro è un dato dell'importazione: aprire o leggere un database non vi scrive nulla.</li>
<li><strong>Un'importazione interrotta dall'arresto dell'applicazione si riprende dalla finestra.</strong> Il comando <code>:resume</code> elenca le importazioni che nulla ha terminato; si sceglie quella da continuare, poi si indicano di nuovo la cartella o i file, come <code>--dir</code> da riga di comando. Il registro del lotto decide che cosa non viene riletto.</li>
<li><strong>Una cartella grande si importa in modalità massiva.</strong> A partire da 200 file, blunderDB scrive con una cache più grande e meno checkpoint. Se il database non contiene ancora alcuna posizione, va oltre: gli indici di ricerca vengono ricostruiti solo alla fine e le scritture non sono più sincronizzate sul disco. Un'interruzione di corrente durante un'importazione del genere può allora danneggiare il database: bisogna ricrearlo e rilanciare l'importazione. Un arresto brusco del programma, invece, lascia solo indici mancanti, che l'apertura successiva ricostruisce (il registro lo segnala).</li>
<li><strong>Ciò che blunderDB non scrive mai</strong>: una fortuna ricalcolata — è letta dal file sorgente o resta sconosciuta — e un rollout che non ha lanciato lui stesso: i dati di rollout di un file <code>.xg</code> non vengono aperti. Sono memorizzati solo i rollout che blunderDB produce (Rollout), accanto all'analisi.</li>
</ul>
<h3>Pannello Match</h3>
<p>Il pannello <strong>Match</strong> (<em>CTRL-Tab</em>) elenca i match importati. Fare doppio clic su un match (o premere <em>INVIO</em>) per navigare tra le sue mosse. Il comando <code>m</code> riprende la navigazione nell'ultimo match visitato.</p>
<p>Il campo filtro, in alto nel pannello (<em>/</em> per andarci, <em>Esc</em> per cancellarlo), mantiene solo le partite in cui un giocatore, l'evento, il luogo, il torneo o la data contiene il testo digitato. Il filtro e l'ordinamento delle colonne sono eseguiti dal database: l'elenco si carica a pagine durante lo scorrimento e il contatore «n / N partite» indica la parte caricata. Correggere un giocatore, una data o un torneo nell'elenco aggiorna solo la riga modificata.</p>
<p>Quando l'elenco è vuoto, il pannello propone <strong>Importa… (Ctrl+I)</strong>; senza un database aperto propone invece <strong>Apri un database…</strong>, insieme a <strong>Torna alla schermata iniziale</strong>. Quando è il filtro di testo ad aver svuotato l'elenco, propone <strong>Cancella il filtro</strong>. I pannelli Stats, Collezioni e Anki vuoti offrono gli stessi pulsanti.</p>
<p>L'utente può:</p>
<ul>
<li>scorrere le mosse di un match usando i tasti <em>SINISTRA</em> e <em>DESTRA</em>,</li>
<li>passare da una partita all'altra con i tasti <em>PageUp</em> e <em>PageDown</em>,</li>
<li>visualizzare l'analisi delle mosse (pedine e cubo) premendo <em>CTRL-L</em>,</li>
<li>alternare tra l'analisi delle mosse di pedine e quella del cubo con il tasto <em>d</em>,</li>
<li>vedere la mossa effettivamente giocata evidenziata nell'analisi,</li>
<li>cercare tra le posizioni del match con il comando <code>ss</code> (es.: <code>ss E&gt;80</code>); <em>Esc</em> riporta poi alla mossa studiata (vedere Pannello Ricerca).</li>
</ul>
<p>L'ultima posizione visitata in ciascun match viene memorizzata e ripristinata automaticamente. Premere <em>CTRL-Tab</em> o eseguire il comando <code>match</code> per mostrare o nascondere il pannello.</p>
<p>Il pulsante <strong>⊕</strong> di una riga arricchisce quell'incontro da un file. Dietro non c'è nulla di nuovo: reimportare lo stesso incontro in un altro formato lo arricchisce già sul posto — l'impronta canonica riconosce che si tratta dello stesso incontro, e le analisi e i commenti del secondo file completano il primo. Ciò che il pulsante apporta è che lo si trova: nessuno indovina che un'importazione è anche un arricchimento. Il resoconto che segue dice quale dei due è avvenuto — «arricchiti: 1» invece di «importati: 1».</p>
<p>Ogni match può essere esportato in trascrizione Jellyfish <code>.mat</code> tramite il pulsante ⬇ dell'elenco dei match o il pulsante <em>.mat</em> della scheda del match.</p>
<p>Un clic su un match apre la sua scheda. La sua scheda <strong>Trascrizione</strong> elenca le mosse partita per partita, e un clic su una mossa vi porta la revisione. Ogni mossa vi porta la sua <strong>gravità</strong>: <code>?</code> per un errore, <code>??</code> per un blunder, un filetto colorato al margine della riga, e il costo della mossa in equity al passaggio del puntatore sul segno. Le soglie sono quelle del database (Configurazione), le stesse con cui contano le statistiche. La mossa è giudicata così come è stata giocata: una stessa posizione giocata due volte nel match riceve due giudizi. Una mossa che l'analisi non valuta non porta alcun segno.</p>
<p>L'intestazione di ogni partita conta i suoi segni, che sia espansa o no: si vede senza aprirla in quale partita si trovano i blunder.</p>
<p>Quando la partita è analizzata, la scheda aggiunge la colonna <strong>MWC</strong>, le probabilità di vincere la partita che la decisione è costata, in percentuale: <code>0</code> per una decisione senza perdita, un trattino per una decisione che l'analisi non valuta o il cui punteggio non ha valore nella tabella di equità della partita (gioco a money, decisione fuori dalle statistiche). Un clic sull'intestazione <strong>MWC</strong> ordina le mosse di ogni gioco dalla perdita più pesante alla più leggera, poi il contrario, poi ripristina l'ordine della partita; le decisioni non valutate passano per ultime, e questo ordinamento sostituisce quello per durata. Sopra i giochi, un riepilogo dà per ogni giocatore la sua perdita di MWC totale (quella dell'elenco delle partite) e il numero di decisioni valutate, con due grafici sullo stesso asse di decisioni di quello delle durate: una barra per decisione, poi la perdita cumulata di ciascun giocatore, che si ferma al suo totale (tratto pieno per il primo giocatore, tratteggiato per il secondo). Una decisione non valutata porta un piccolo segno alla base della barra, senza altezza. Passare sopra una decisione, o scorrerle con le frecce (i tasti Inizio e Fine portano alle estremità), la segna sui due grafici e sulla sua riga della trascrizione e mostra il gioco, la mossa, il giocatore e la perdita; un clic, o Invio, porta lì la revisione e fa scorrere la trascrizione fino alla riga. Lo stesso vale per il grafico delle durate. Fuori dall'interfaccia, <code>match --format json</code> aggiunge <code>decision_losses</code> (una voce per mossa, con <code>mwc_loss</code> pari a <code>null</code> per una mossa non valutata), <code>--format text</code> una riga « MWC loss » per mossa valutata e <code>--format summary</code> il totale per giocatore.</p>
<p><strong>Difficoltà ed errori evitabili.</strong> Una perdita dice quanto è costata una mossa, non se era difficile trovare quella giusta. La colonna <strong>Diff.</strong>, accanto a <strong>MWC</strong>, dà la <em>difficoltà</em> della decisione: la perdita di probabilità di vincere la partita che un <em>giocatore di riferimento</em> subirebbe in media nella stessa posizione. Questo giocatore di riferimento non gioca alla perfezione: sceglie ogni opzione con una probabilità tanto più bassa quanto più costa, π(i) proporzionale a exp(−Δᵢ/τ), dove Δᵢ è il costo dell'opzione i rispetto alla migliore, in equity normalizzata. La difficoltà vale d = Σ π(i)·Δᵢ, convertita in MWC al punteggio e al cubo della decisione come la perdita: è espressa nella stessa unità, una percentuale di probabilità di vincere la partita. Le opzioni sono i candidati dell'analisi per una mossa di pedine; non raddoppio e raddoppio (contro la migliore risposta) per chi ha il cubo; accetto e passo per chi riceve il raddoppio. La temperatura τ vale 0,025: la decisione a due opzioni più difficile è quella con uno scarto di circa 0,032, e una posizione con cinque candidati distanziati di 0,01 dà l'errore medio di un giocatore di circa PR 6; è un a priori di giocatore forte, fissato prima di esaminare qualsiasi risultato.</p>
<p>Come leggerla: una decisione ovvia, o forzata, ha una difficoltà vicina a <code>0</code> senza che alcuna soglia la distingua; una decisione serrata, in cui più opzioni stanno a pochi millesimi, ne ha una grande. Una perdita molto superiore alla difficoltà è un errore che la posizione non giustificava; una perdita vicina alla difficoltà, un errore che farebbe anche un buon giocatore. Una decisione è un <strong>errore evitabile</strong>, contrassegnato da un <code>!</code> accanto alla sua perdita e da un punto sopra la sua barra, quando la sua perdita raggiunge la soglia di errore della base e la sua difficoltà non ne supera un decimo: su una decisione a due opzioni, il giocatore di riferimento non la commetterebbe una volta su dieci. Nel grafico per decisione, la difficoltà è un tratto orizzontale su ogni barra: una barra che sale molto sopra il suo tratto segnala un errore evitabile.</p>
<p>La tabella sopra i grafici aggiunge, per ogni giocatore e sulle decisioni che hanno una perdita e una difficoltà: la <strong>difficoltà</strong> totale, l'<strong>eccesso</strong> Σ(perdita − difficoltà), in MWC, ciò che il giocatore ha perso oltre il giocatore di riferimento (negativo quando ha fatto meglio), il <strong>rapporto</strong> perdita totale / difficoltà totale (1: gioca come il giocatore di riferimento; 2: perde il doppio) e il numero di <strong>errori evitabili</strong>. Il rapporto non è dato quando la difficoltà totale è sotto lo 0,5 %: su una partita troppo facile misurerebbe solo rumore, l'eccesso resta.</p>
<p>Perché guardarla: il PR e la perdita MWC mescolano due cose, la qualità del gioco e la difficoltà delle posizioni incontrate. Un avversario che crea posizioni complesse fa salire l'errore dell'altro; una partita di corsa pura lo fa scendere. La difficoltà corregge entrambi gli effetti: l'eccesso e il rapporto confrontano il giocatore con ciò che ci si poteva aspettare nelle <em>sue</em> posizioni. Per lo studio, ordina gli errori: gli errori evitabili sono disattenzioni o una regola saputa male, da rivedere per primi e spesso facili da correggere; le perdite su decisioni difficili sono lavoro di fondo (rollout, principi, posizioni di riferimento) e pesano meno nel giudizio su una partita.</p>
<p>Incertezza e limiti: la difficoltà dipende da un modello di giocatore e da τ, fissati una volta; cambiare l'uno cambia tutte le cifre. Conosce solo i candidati che l'analisi ha conservato (pochi in XG, secondo i filtri in GNU Backgammon): le opzioni assenti non pesano nulla e la difficoltà è allora un minorante. Eredita l'errore dell'analisi stessa, soprattutto a bassa profondità. Su una sola partita, l'eccesso e il rapporto poggiano su poche decisioni e variano molto da una partita all'altra: indicano una tendenza, non una classifica. La difficoltà è calcolata solo per le decisioni la cui perdita è valutata; altrove vale un trattino. Fuori dall'interfaccia, <code>match --format json</code> riporta <code>difficulty</code> e <code>avoidable</code> su ogni voce di <code>decision_losses</code> e il riepilogo per giocatore sotto <code>difficulty_summary</code>; <code>--format text</code> aggiunge le righe «Difficulty» e «Avoidable error», <code>--format summary</code> la difficoltà, l'eccesso, il rapporto e il numero di errori evitabili.</p>
<p><strong>Bilancio del match.</strong> Sopra i grafici, un riquadro risponde, per ogni giocatore, alle tre domande che ci si pone dopo un match: che cosa rivedo, ho perso per i dadi o per il gioco, e i miei errori vengono dalla fretta o da una lacuna? Le soglie sono state fissate prima di esaminare qualsiasi risultato.</p>
<ul>
<li><strong>PR e perdita di MWC (eq. 7 punti), con il loro intervallo al 95 %</strong>, calcolato ricampionando le partite del match. Servono due partite; un match di una sola partita mostra un trattino. Un intervallo ampio dice che un match non basta per giudicare un livello: confrontatelo con gli altri match invece di concludere su un solo valore.</li>
<li><strong>Risultato corretto per la fortuna</strong>, per un match concluso. Il <em>risultato</em> è l'esito (100 % vinto, 0 % perso) meno le probabilità iniziali (50 % sullo 0-0); la <em>fortuna netta</em> somma la fortuna dei vostri tiri meno quella dei tiri avversari, ogni tiro convertito in MWC al punteggio e al cubo della sua posizione, come una perdita. Il risultato corretto è il risultato meno la fortuna netta. Se è positivo in un match perso, hanno deciso i dadi; negativo, il gioco. Lo <em>scarto degli errori</em> (perdita dell'avversario meno la vostra) è ciò che il risultato corretto stima: le due cifre coincidono quando la fortuna è completa. La copertura è indicata quando alcuni tiri non hanno una fortuna misurata (un file senza fortuna, un tiro non analizzato); senza alcun tiro misurato, o per una partita a denaro, la riga non compare.</li>
<li><strong>Tre decisioni da rivedere</strong>: tra i vostri errori (soglia della base), quelli con la <em>parte evitabile</em> della perdita, perdita − difficoltà, più grande — cioè la perdita moltiplicata per il suo carattere evitabile. Un errore che avrebbe commesso anche il giocatore di riferimento non vale una sessione: si rivede prima ciò che era alla portata. Un clic sulla decisione la mostra, come una riga dell'estratto.</li>
<li><strong>Errori affrettati e ponderati</strong>, quando il match ha conservato la durata delle decisioni: un errore giocato più in fretta della mediana delle vostre decisioni dello stesso tipo (pedine o cubo) in questo match è affrettato, gli altri ponderati. La mediana è la vostra, in questo match: non dipende dal ritmo, e senza legame tra velocità ed errore gli errori si dividerebbero metà e metà. Una maggioranza affrettata richiede disciplina (rallentare su queste posizioni); una maggioranza ponderata, conoscenza (studiare la famiglia di posizioni).</li>
</ul>
<p><code>match --format summary</code> stampa lo stesso bilancio, e il server lo serve tramite <code>/v1/stats.matchReview</code>.</p>
<p>Quando la partita ha conservato la durata delle sue decisioni (una partita giocata contro un bot), la scheda aggiunge tre colonne allineate sulle cifre: <strong>Cubo</strong> (la decisione di cubo, presa prima del lancio o sulla riga del cubo stessa), <strong>Gioco</strong> (la mossa di pedine) e <strong>Orologio</strong>, il tempo che il giocatore di turno ha consumato dall'inizio della partita, questa mossa compresa, qualunque sia l'ordinamento. Sotto una cadenza, la colonna diventa <strong>Orologio residuo</strong> e dà il tempo che resta sull'orologio del giocatore dopo la mossa, calcolato come lo conta l'arbitro del Duello (ritardo per turno, riserva azzerata al superamento). Le informazioni della partita e la sua intestazione ricordano la cadenza (preimpostazione, ritardo, comportamento al superamento) e la banca del tempo di ciascun giocatore, riportata al punteggio iniziale quando la riserva è contata per punto rimanente. Un clic sull'intestazione <strong>Gioco</strong> ordina le mosse di ogni gioco dalla più lunga alla più breve, poi dalla più breve alla più lunga, poi ripristina l'ordine della partita; una casella senza durata (nessuna decisione di cubo in quel turno, o mossa giocata dal solo Arbitro) porta un trattino, e quelle mosse vanno per ultime. L'Orologio parte da zero dalla prima mossa. Sopra i giochi, un riepilogo dà per ogni giocatore il totale, la media per mossa di pedine e per decisione di cubo e, se la partita ha una cadenza, un segno per il giocatore la cui riserva si è esaurita per prima (il Duello registra solo quello); un grafico colloca la durata di ogni decisione nel corso della partita. In revisione, la durata della decisione giocata si legge discretamente sotto l'analisi. La ricerca la filtra con <code>tm&gt;30</code> (in secondi), che si combina con <code>E&gt;x</code>: <code>s tm&gt;30 E&gt;80</code> trattiene le mosse a lungo meditate e comunque sbagliate, essendo durata ed errore quelli della stessa mossa giocata. La scheda Statistiche, sotto gli errori ricorrenti, incrocia tempo ed errore: per ogni giocatore e ogni fascia di durata nota (meno di 5 s, da 5 a 15 s, da 15 a 30 s, più di 30 s), il numero di decisioni, l'errore medio e la quota di blunder. Una decisione il cui errore non è registrato è contata senza entrare nella media.</p>
<p>Un match giocato qui, nato da un Duello, porta la sua origine in testa alla trascrizione, su una riga, anche se non ha alcuna mossa: «Giocato qui», poi, se del caso, «Perso per tempo» con il giocatore la cui riserva si è esaurita sotto una cadenza che fa perdere il match, o «Interrotto prima della fine» per un Match che un Duello interrotto ha lasciato incompiuto, la cadenza, il giocatore la cui riserva si è esaurita per prima e il livello del Bot con la versione di gammonNet di cui ha giocato la politica. Un clic apre la partenza (la posizione iniziale o lo XGID scelto), il seme dei dadi rivelato e il suo SHA-256, da confrontare con l'impronta pubblicata alla creazione del Duello: se coincidono, il seme permette di ricalcolare ogni lancio senza fidarsi di blunderDB. Un match che non è stato giocato qui non ha origine e non mostra questa riga.</p>
<p>La scheda <strong>Infos</strong> della scheda del match riprende l'intestazione del match. Vi aggiunge ciò che il file di origine dice dei giocatori e della sessione, quando lo dice — un file eXtreme Gammon lo dice sempre: il punteggio Elo di ciascun giocatore con la sua esperienza tra parentesi, il trascrittore, le regole Jacoby e Beaver di una partita libera e il programma che ha scritto il file. Queste informazioni vengono esportate con il match. Reimportare un file già presente le dà al match che non le aveva, senza sostituire nulla di ciò che già porta. Anche il comando <code>match</code> della riga di comando le mostra. I commenti di intestazione e di chiusura del match di un file eXtreme Gammon diventano il commento del match, firmato con il nome del suo trascrittore, o <code>XG</code> se il file non ne indica uno; l'orologio e la tabella di equity non vengono importati. La scheda mostra questo autore accanto al commento, e l'esportazione lo copia con esso; modificare il commento lo firma con <strong>Il tuo nome</strong> (impostazioni).</p>
<p>Un match trascritto da un video porta la sua <strong>sorgente</strong>: un URL http(s) (YouTube, per esempio) o il percorso di un file. La riga <strong>Video</strong> della scheda <strong>Info</strong> la modifica: digitare un URL o un percorso e premere <em>INVIO</em>, <strong>File…</strong> per scegliere un video, <strong>Scollega</strong> per togliere la sorgente. Un match che ne ha una mostra l'icona 🎞 nella barra della sua scheda, e ogni decisione che porta un segno del video mostra la stessa icona nella sua riga della scheda <strong>Trascrizione</strong>. Cliccare l'icona, o premere <em>v</em> quando la decisione è quella in revisione, porta il video un secondo prima del lancio dei dadi: per un file si apre un riquadro video sopra la scheda; per una sorgente YouTube il browser apre il link con il marcatore temporale. Un file mancante si ricollega dal riquadro, e un formato che il webview non riproduce vi è segnalato con i pacchetti da installare (vedere Download e installazione). Un match senza sorgente non cambia.</p>
<p>Il pulsante <strong>Unisci giocatori</strong> della barra degli strumenti del pannello apre una finestra che elenca tutti i nomi dei giocatori del database con il loro numero di match: selezionare le varianti di ortografia di uno stesso giocatore, scegliere il nome canonico da conservare, quindi unire. L'unione crea un alias per variante: i match conservano i nomi dei loro file, ma le statistiche, la tabella Giocatori e la ricerca <code>pl"…"</code> leggono tutte le varianti come un solo giocatore, e le importazioni successive registrano il nome canonico. Rimuovere l'alias nella scheda <strong>Corpus</strong> delle impostazioni annulla l'unione.</p>
<p>Quando un match è aperto, una <strong>barra delle informazioni</strong> compare sopra il tavoliere: ricorda i giocatori presenti (<em>giocatore 1</em> contro <em>giocatore 2</em>) nonché il contesto del match (evento, luogo, turno, data e lunghezza del match, quando queste informazioni sono disponibili). Questa barra viene mostrata anche al di fuori della modalità match: quando una posizione studiata (proveniente da una ricerca, da una collezione o da un accesso diretto) proviene da uno o più match, ne indica la <strong>provenienza</strong> — il primo match interessato e, se del caso, un badge « +N » che elenca gli altri al passaggio del mouse. Una posizione importata da sola, che nessun match referenzia, non mostra nulla.</p>
<p>Le schede <strong>Ricerca</strong> e <strong>Eval</strong> sostituiscono la scacchiera con una scacchiera di lavoro: un banner nella parte alta della scacchiera lo segnala («Scacchiera di ricerca», «Scacchiera di valutazione») e la barra informativa viene nascosta finché descriverebbe una posizione che non è sullo schermo. Il ritorno all'analisi ripristina la posizione studiata.</p>
<p>All'apertura di un database contenente match, il pannello <strong>Match</strong> viene mostrato subito e la revisione inizia direttamente sulla prima posizione, così da cominciare immediatamente la navigazione.</p>
<div class="admonition note">
<p>Un database può essere aperto in scrittura da una sola finestra alla volta. Se si apre un database già aperto in un'altra finestra di blunderDB, esso si apre in <strong>sola lettura</strong> : la navigazione, la ricerca e l'analisi restano possibili, ma qualsiasi modifica è disattivata e la barra del titolo mostra « [sola lettura] ».</p>
</div>
<div class="admonition tip">
<p>Fare riferimento a Scorciatoie da tastiera per le scorciatoie disponibili.</p>
</div>
<h3>Pannello Trascrizione</h3>
<p>Il pannello <strong>Trascrizione</strong> (<em>CTRL-MAIUSC-T</em>, comando <code>transcribe</code> o <code>tr</code>) serve a digitare un match che si ha sotto gli occhi — un foglio di match, una registrazione video — per farne un match della libreria. Ciò che si digita è una <strong>bozza</strong>: vive nella base, si chiude e si riapre, e non entra né nelle statistiche né nelle ricerche finché non è stata salvata come match.</p>
<p>Il pannello si apre sulla <strong>lista delle bozze</strong> del database: ultima modifica, giocatori, lunghezza, numero di azioni e la partita già prodotta (<code>#</code> seguito dal suo identificativo) oppure la dicitura «nessuna partita». Un clic apre una bozza, e il pulsante <strong>Bozze</strong> della barra vi riporta. Il pulsante <strong>Nuova trascrizione</strong> apre il modulo di creazione.</p>
<p>La lista si percorre anche da tastiera: <em>GIÙ</em> e <em>SU</em> (o <em>j</em> e <em>k</em>) spostano l'evidenziazione, <em>INVIO</em> apre la bozza evidenziata, <em>n</em> apre il modulo. La prima bozza è evidenziata all'apertura ed è quella modificata più di recente: riprendere il lavoro di ieri costa dunque due tasti, <em>CTRL-MAIUSC-T</em> e poi <em>INVIO</em>.</p>
<p>Il modulo chiede una cosa sola: la <strong>lunghezza del match</strong>. Il valore <code>0</code> indica una partita a soldi e fa comparire le caselle <em>Jacoby</em> e <em>Beaver</em>. Il campo si apre sulla lunghezza dell'ultima bozza modificata, oppure su 7 quando la base non ne contiene alcuna. I nomi dei giocatori non vengono chiesti: la bozza designa i lati come <em>Giocatore 1</em> e <em>Giocatore 2</em>, e l'elenco mostra «Senza nome».</p>
<p>Tutto ciò che si deduce da quanto è stato digitato — la lunghezza della partita (o «Denaro»), il punteggio, la dicitura <em>Crawford</em> quando la partita in corso lo è, il numero della partita, lo stato del cubo — il suo valore, al centro o a nome di chi lo possiede — e il campo di turno — compare nella <strong>barra della partita</strong>, sopra la tavola: è lì che lo sguardo si trova già quando ci si chiede chi sia di turno. L'azione attesa, invece, è scritta per esteso nella <strong>barra di stato</strong>: «dadi di Kévin», «risposta di Alice al raddoppio».</p>
<p>La <strong>barra della bozza</strong>, in cima al pannello, porta solo i gesti che fanno uscire la bozza da sé stessa, le due frecce di annullamento e l'orientamento del tavoliere.</p>
<p><strong>Il giocatore 1 resta in basso sul tavoliere</strong>, chiunque sia di turno. Un match che si trascrive è una partita che si svolge: il turno cambia a ogni mezza mossa, e seguirlo capovolgerebbe il tavoliere da un turno all'altro — le pedine appena guardate passerebbero in alto e l'occhio rifarebbe il percorso a ogni lancio. Il turno si legge dai <strong>dadi</strong>, che cambiano lato. Il pulsante ⇅ della barra capovolge il tavoliere e mostra il giocatore 2 in basso; non modifica la bozza, e alla chiusura la visualizzazione torna al verso giusto. Da non confondere con il pulsante <em>Inverti i giocatori</em> del riquadro Metadati, che scambia i due giocatori nel documento stesso.</p>
<p>Il pulsante <strong>Metadati</strong> della barra apre l'intestazione della bozza, in qualsiasi momento: i nomi dei due giocatori — completati automaticamente dai giocatori della base —, l'evento, il luogo, il turno, la data (quella odierna per impostazione predefinita), il trascrittore (l'utente della base per impostazione predefinita) e il torneo a cui il match sarà collegato al momento del salvataggio. Nessun campo è obbligatorio: una bozza senza nomi si salva e si esporta lo stesso, con intestazioni vuote. Il pulsante <strong>Invertire i giocatori</strong> scambia i due nomi, assegna tutte le azioni al campo opposto e gira la damiera: è lo stesso match, letto dall'altro lato.</p>
<p>La <strong>lunghezza del match</strong> si cambia in questo stesso pannello, in qualsiasi momento: il punteggio, la partita Crawford e il riferimento — le partite a soldi quando la lunghezza vale <code>0</code>, e compaiono allora le caselle <em>Jacoby</em> e <em>Beaver</em> — sono ricalcolati da un capo all'altro della bozza, e le azioni registrate dopo la vittoria sono segnalate «oltre la fine» senza che nessuna venga eliminata. La lunghezza fa parte dell'identità di una posizione: dopo un salvataggio, cambiarla e salvare di nuovo scrive posizioni nuove, da analizzare, e le vecchie spariscono non appena più nulla le trattiene.</p>
<p>Sotto la barra, la bozza occupa tre regioni: la <strong>tavolozza</strong> dei bersagli del mouse, le <strong>mosse candidate</strong> e la <strong>trascrizione</strong>. Si dispongono secondo la larghezza del pannello. In un pannello largo — il dock in basso — le tre stanno una accanto all'altra, con la tavolozza a sinistra. In un pannello medio, le candidate occupano la parte alta e la trascrizione viene accanto al triangolo dei lanci, nello spazio che questo lascia alla sua destra. In un pannello stretto le tre si susseguono: candidate, tavolozza, trascrizione — lì il triangolo e la trascrizione non stanno affiancati senza amputare alla trascrizione la seconda colonna, e basta allargare il dock di qualche decina di pixel per riunirli. In ogni caso la regola è la stessa: nulla si interpone fra le due caselle del lancio e la prima riga delle candidate, e almeno cinque candidate si leggono senza far scorrere alcunché.</p>
<p>La tavolozza mostra i due dadi man mano che vengono inseriti; un clic su di essi li cancella, come <em>BACKSPACE</em>. Una partita comincia con la sua prima mossa, giocata dal vincitore del tiro d'apertura: i suoi due dadi si digitano come quel tiro, il dado del giocatore 1 e poi quello del giocatore 2, e il più alto dà la mossa al suo campo, che gioca entrambi i dadi. La mossa si sceglie poi tra i candidati, come ogni altra. Le parità, ritirate al tavolo, non si trascrivono; una prima mossa digitata come doppio è registrata così com'è e segnata «dadi incoerenti», poiché nessun tiro d'apertura è un doppio, e un raddoppio del cubo prima della prima mossa è segnato «azione di cubo impossibile». Un'azione di cubo registrata dopo la fine di una partita porta lo stesso segno: resta nella partita terminata, non ne apre una nuova e si elimina a mano.</p>
<p>Non appena cade il secondo dado, tutte le <strong>mosse legali</strong> del lancio vengono elencate, classificate dal motore incorporato, la prima preselezionata e le sue frecce poste sulla tavola. La lista dà la mossa, la sua equità e il suo scarto dalla migliore: trascrivere significa riconoscere la mossa che si è vista giocare, non giudicarla — per questo c'è il pannello <strong>Valutazione</strong>. Questa classifica è una valutazione: viene mostrata, non viene mai scritta nel database. Quando il motore non è disponibile, le mosse sono elencate senza classifica e la lista lo dice in testa.</p>
<p>La <strong>rotellina</strong> seleziona la candidata successiva o precedente, sia sopra la lista sia sopra la tavola: lo sguardo resta sulla tavola e le frecce scorrono, il che riconosce una mossa più in fretta della lettura della sua notazione. Un clic su una riga la seleziona, un doppio clic la convalida.</p>
<p>Il triangolo dei ventuno tiri sta sotto le due caselle del tiro, accanto alla tastiera e non al suo posto: due cifre restano due volte più rapide di un clic, e il triangolo è lì per chi trascrive con la mano sul mouse. Una casella per tiro, mai due: 3-1 e 1-3 sono lo stesso tiro.</p>
<p>Una mossa giocata sul tavoliere risparmia la lettura dei dadi. Finché nessun dado è stato inserito, un clic su una pedina e poi sulla sua destinazione — o un trascinamento dall'una all'altra — gioca la mossa sul tavoliere, vincolata alle mosse legali; le destinazioni offerte dalla pedina scelta si illuminano. I due dadi si deducono dai passi: giocare 13/7 e poi 8/7 dice 6-1 senza che sia stata digitata una cifra, e l'azione viene registrata non appena la mossa è completa. Backspace annulla l'ultimo passo, una cifra abbandona la mossa e torna all'inserimento dei dadi, e <em>Recommencer</em>, in cima al menu del clic destro, la ricomincia. Quando più lanci producono la stessa mossa — un'uscita che più dadi coprono, un dado non giocabile — nulla viene registrato e il triangolo lascia cliccabili solo quei lanci: il lancio non viene mai indovinato al posto di chi guarda la partita.</p>
<p>Con i due dadi inseriti, anche il tavoliere gioca, vincolato alle mosse legali di quel lancio — alla fine del documento come su un'azione rivista, di cui il cursore ha caricato i dadi. Ogni passo giocato lascia nella lista solo i candidati che lo contengono, il primo dei quali preselezionato: è il gesto della mossa lontana, là dove scendere al dodicesimo candidato costa tredici tasti. Una mossa legale completa viene registrata subito, con i dadi così come sono stati digitati; su un'azione rivista, la sostituisce.</p>
<p>Una mossa illegale si trascrive così come è stata giocata, senza pulsanti né cambio di modalità. Con i dadi inseriti, un trascinamento che nessuna mossa legale offre posa la pedina dove viene rilasciata — anche da un punto da cui non parte alcuna mossa legale, purché porti una pedina del giocatore di turno. La mossa esce allora dalle regole: il resto si gioca liberamente, con il clic come con il trascinamento, la lista dei candidati lascia il posto a una riga che lo ricorda, e nulla viene registrato prima di INVIO, che scrive i dadi inseriti, i passi e il tavoliere ottenuto. Backspace annulla l'ultimo passo; annullare l'unico passo fuori dalle regole restituisce la lista. Senza dadi inseriti, il trascinamento resta vincolato: una mossa illegale non dice quale lancio l'abbia prodotta.</p>
<p>La mossa si digita anche da tastiera, nella trascrizione. Un doppio clic sulla cella di una mossa — o di una danza, di una mossa non registrata — la trasforma in un campo, precompilato con la sua notazione. Lì si digita solo la mossa, <code>13/7 8/7*</code>, <code>bar/22</code> o <code>6/off</code>: i dadi sono quelli della cella. INVIO la registra al posto della mossa scritta, ESC richiude la cella senza scrivere nulla, e un testo che non indica alcuna mossa lascia il campo aperto. La cella tratteggiata dell'inserimento in corso si apre allo stesso modo, non appena i suoi due dadi sono inseriti.</p>
<p>Una mossa immessa con il trascinamento libero o con la notazione che risulti legale resta una mossa ordinaria — il confronto si fa sulla tavola ottenuta, mai sulla provenienza del gesto; altrimenti viene marcata «mossa illegale» nella trascrizione, e l'esportazione <code>.mat</code> avverte prima di scrivere il file, senza mai rifiutare.</p>
<p>Sulla riga dei dadi, la fila <strong>Raddoppiare</strong>, <strong>Accettare</strong>, <strong>Passare</strong>, <strong>Abbandonare</strong> porta al mouse i quattro gesti del cubo: sono, insieme ai due dadi, le cinque risposte possibili a una sola domanda — che cosa ha fatto il campo di turno? Dice a chi tocca: il campo di turno annuncia — raddoppiare, abbandonare — oppure il campo avverso risponde — accettare, passare; mai tutti e quattro insieme, e un pulsante il cui gesto non risponderebbe a nulla resta spento. La tastiera, invece, non rifiuta mai nulla: un pulsante spento è un bersaglio che non si offre, non un gesto vietato. «Abbandonare» non registra ancora nulla: la fila diventa i tre livelli — semplice, gammon, backgammon — e «Annulla», che raddoppia il tasto ESC. Il cubo disegnato sulla tavola è il secondo bersaglio di questi gesti: un clic su di esso propone un raddoppio. Davanti a un'offerta non risponde — accettare e passare sono due risposte simmetriche e vivono insieme nella fila, un clic ciascuna.</p>
<p>La <strong>barra di stato</strong> dice in una parola che cosa la bozza attende: la danza registrata d'ufficio, la prima mossa di una partita, la risposta attesa a un raddoppio, il livello atteso dopo un abbandono, la correzione sul posto, la mossa «da rivedere» il cui lancio è cambiato. Vi risponde anche ai gesti che non hanno nulla da fare — «niente da annullare», «nessuna azione sotto il cursore» — per un secondo e mezzo. L'incoerenza che un'azione ha lasciato dietro di sé è invece segnalata in testa alla trascrizione, là dove si trova la cella difettosa.</p>
<p>Una partita finisce con un passo, con un abbandono o con l'uscita della quindicesima pedina (semplice, gammon o backgammon, moltiplicato per il valore del cubo). Il punteggio, la partita Crawford e la fine dell'incontro compaiono allora nella barra della partita, e si attende la prima mossa della partita successiva.</p>
<p>Il punteggio di una partita è quello che danno le partite precedenti, salvo se al tavolo ne è stato dichiarato un altro. Un doppio clic sul punteggio nell'intestazione di una partita, nella trascrizione, lo trasforma in un campo precompilato: vi si digita il punteggio al quale la partita è stata giocata — <code>3-2</code>, <code>3–2</code> o <code>3 2</code> —, INVIO lo registra, ESC richiude il campo senza scrivere nulla, e un campo svuotato poi confermato torna al punteggio derivato. La partita è giocata a quel punteggio: la partita Crawford, la fine dell'incontro e le partite seguenti ne discendono, e sia l'incontro salvato sia il file <code>.mat</code> lo riportano. Un punteggio diverso da quello derivato è segnalato, il suggerimento indica quello derivato, e la prima azione della partita porta l'incoerenza «punteggio dichiarato incoerente». Nel gioco a soldi non c'è punteggio da dichiarare.</p>
<p>La trascrizione occupa la metà destra del pannello: una colonna per giocatore, una riga per turno, l'azione di cubo e la fine della partita nella colonna di chi agisce. La cella del cursore è incorniciata; spostare il cursore riporta la tavola alla posizione dell'azione mirata e mostra le sue candidate, con la mossa registrata selezionata. Un'incoerenza (mossa illegale, doppio turno, azione di cubo impossibile, azione oltre la fine dell'incontro, dadi incoerenti, mossa non registrata, punteggio dichiarato incoerente) decora la sua cella e viene nominata in un suggerimento. La mossa non registrata è il caso di un file <code>.mat</code> riletto: gnubg vi scrive <code>???</code> quando non ha conservato la mossa giocata, il lancio è noto e la mossa no, e mettere il cursore su quella cella propone le mosse di quel lancio per completarla. Un doppio turno lascia una cella vuota, incorniciata da un tratteggio, nella colonna del campo a cui manca il turno: il cursore vi si ferma, un clic ve lo porta, ed è lì che si digita il turno mancante — una decisione eliminata, per esempio. Le partite si ripiegano; quella del cursore è aperta.</p>
<p><strong>Ciò che si sta digitando è disegnato nella trascrizione</strong>, tratteggiato, nel punto esatto in cui sarà scritto: i dadi man mano che cadono, la notazione della mossa appena è selezionata, e nella colonna del campo a cui l'azione appartiene. Una correzione copre la cella che sostituisce, un inserimento apre una cella tra le sue due vicine, una digitazione nuova compare in fondo alla partita in corso. Nulla è scritto nella bozza prima della convalida; ciò che si legge e ciò che il documento dirà non divergono mai.</p>
<p>Correggere è digitare sulla cella in cui ci si trova. Con il cursore su un'azione, una cifra ne ricomincia il lancio sul posto, e anche le quattro azioni di cubo valgono come correzioni: su un rifiuto, <em>t</em> — o il pulsante <strong>Accetta</strong> — scrive un'accettazione <strong>al posto</strong> del rifiuto, senza doverlo cancellare e poi inserire. La partita riprende allora il suo corso: si apre una cella subito dopo l'accettazione, dalla parte di chi ha raddoppiato, e il seguito della partita vi si digita come al solito, inserito davanti alla prima mossa della partita successiva, finché la partita non termina. Gli stessi tasti riempiono la cella che un inserimento ha appena aperto: <em>i</em> e poi <em>d</em> inserisce un raddoppio davanti all'azione mirata. Quella prima mossa conserva il punteggio con cui iniziava la sua partita, ora dichiarato: se la fine della partita ripresa ne dà un altro, la differenza viene segnalata.</p>
<p>Tornare sull'<strong>ultima</strong> azione significa tornare là dove si scrive la trascrizione. Una cifra digitata su di essa ne corregge ancora il lancio, ma una volta ridigitato quel lancio, la cifra successiva la convalida e apre la decisione seguente; INVIO la convalida allo stesso modo. Le decisioni successive si aggiungono allora in coda, come la prima volta.</p>
<p>Digitare <strong>un altro tiro</strong> sulla <strong>prima mossa</strong> di una partita decide di nuovo chi inizia: il dado del giocatore 1 si digita per primo, quello del giocatore 2 dopo, e vince il più alto — dado grande prima, la mossa va al giocatore 1, in basso sul tavoliere; dado piccolo prima, al giocatore 2, in alto. Il primo candidato del nuovo tiro è preselezionato e la mossa è da rivedere. Ridigitare <strong>lo stesso tiro</strong>, in un ordine o nell'altro, non cambia nulla. Per dare la prima mossa all'altro giocatore senza cambiare tiro si usa <em>s</em>: l'ordine dei dadi segue la parte, e una prima mossa giocata dalla parte che quell'ordine non designa è segnalata «dadi incoerenti». Il resto della partita mantiene le sue parti.</p>
<p>Un <strong>inserimento in mezzo al documento continua a inserire</strong>: la convalida apre una cella vuota di seguito, e l'azione successiva si inserisce a sua volta invece di sovrascrivere quella dopo. È questo che permette di recuperare tutta la fine di una partita — un rifiuto che avrebbe dovuto essere un'accettazione — senza perdere ciò che è già stato digitato della partita seguente. La fine della partita, o lo spostamento del cursore, mette fine all'inserimento: il cursore si posa allora sulla prima mossa della partita successiva.</p>
<p><strong>Canc</strong> (o <em>x</em>) toglie la decisione in corso di modifica e arretra sulla precedente, pronta per essere corretta: su una cella scritta, l'azione scompare; su un inserimento aperto o un lancio digitato in fondo al documento, è l'immissione a essere abbandonata. Premendo Canc più volte si risale così la trascrizione cancellando. Le azioni successive mantengono il loro campo, e il doppio turno lasciato da un'eliminazione è segnalato senza che il cursore vi sia riportato.</p>
<p>Un clic destro su una cella apre le correzioni di quell'azione — inserire prima, inserire dopo, eliminare, cambiare campo — e vi porta il cursore per strada; sono gli stessi gesti dei tasti <em>i</em>, <em>a</em>, <em>x</em> e <em>s</em>, e il menu del browser viene soppresso soltanto lì. Altrove non hanno pulsanti: un pulsante che agisse sull'«azione sotto il cursore» mirerebbe a una cella che si può non vedere, mentre il clic destro nomina la propria.</p>
<p>La barra della bozza porta le sue due sole uscite. «<strong>Termina</strong>» (CTRL-INVIO) scrive la partita nella biblioteca e libera la bozza; l'analisi delle sole posizioni nuove parte subito, con il suo avanzamento e la sua annullabilità nella barra di stato. «<strong>Abbandona</strong>» elimina la bozza senza partita; la conferma viene chiesta solo per una bozza mai terminata, poiché porta via tutto ciò che vi è scritto. Accanto, la barra dice che cosa farà Termina — una nuova partita, o la sostituzione della partita #<em>n</em>. Non dice nulla della salvaguardia della bozza stessa: viene scritta nella base dopo ogni azione, e tornare all'elenco la lascia da riprendere più tardi.</p>
<p>«<strong>Testo .mat</strong>» apre il file Jellyfish così come verrebbe scritto, in una finestra abbastanza larga perché le sue colonne restino allineate, con un pulsante per copiarlo. «<strong>Esportare .mat</strong>» scrive quello stesso file su disco. Le due frecce <strong>↶</strong> e <strong>↷</strong> annullano e ripristinano, come <em>CTRL-Z</em> e <em>CTRL-MAIUSC-Z</em>.</p>
<p>Se l'analisi di un match trascritto è stata interrotta — l'applicazione chiusa durante il lotto —, la barra di stato lo segnala alla successiva apertura della base e propone di terminarla. Di questa interruzione non viene conservato nulla: la proposta ritorna finché restano posizioni da analizzare, e il lotto riavviato riguarda solo questo match, mai l'intera biblioteca.</p>
<p>Una bozza che porta incoerenze viene comunque terminata, dopo un avviso: nulla viene rifiutato. Una mossa illegale viene esportata così come è stata giocata, con l'avviso che gnubg e XG la segnaleranno («Invalid move») e divergeranno in seguito.</p>
<p>Per correggere un match della libreria, il pulsante ⌨ dell'elenco dei match o «<strong>Modifica la trascrizione</strong>» della sua scheda apre una bozza a partire da quel match — o riapre quella già aperta su di esso: una sola bozza per match. Terminare questa bozza sostituisce il match con lo stesso identificatore; le posizioni delle azioni invariate conservano i loro commenti, le loro analisi e le loro carte. Un match importato (XG, GnuBG, BGF) porta analisi e commenti che un <code>.mat</code> non porta: prima di aprire, una finestra di dialogo dice fino a quanti, e che terminare la bozza può farli perdere.</p>
<p>Un match si trascrive anche <strong>da un video</strong>. Il pulsante <strong>Video</strong> della barra della bozza propone <strong>File…</strong> per scegliere un video sul disco, <strong>Link YouTube…</strong> per incollare un indirizzo e <strong>Stacca</strong> per rimuovere la sorgente. Finché nessuna sorgente è allegata, il pannello resta come descritto sopra: né riquadro, né tasto in più. Una volta allegata la sorgente, sopra la trascrizione si apre un riquadro video; la sua altezza si regola trascinando la barra sottostante e resta la stessa da una sessione all'altra. Un file introvabile si ricolloca dal riquadro con <strong>Scegli il file…</strong>.</p>
<p>Con un video, ogni mossa <strong>nuova</strong> porta dei <strong>marcatori</strong>: l'istante del lancio, fissato dal primo tasto dei dadi, e l'istante della mossa, fissato dalla convalida, entrambi letti dal video nel momento del gesto. Una correzione sul posto non tocca i marcatori. Solo una convalida esplicita fissa l'istante della mossa: <em>INVIO</em>, il doppio clic su un candidato o la mossa completata sulla scacchiera. La cifra del lancio successivo convalida anch'essa la mossa, ma non fissa alcun istante: la mossa resta senza istante di mossa anziché riceverne uno falso, e la barra di stato lo ricorda. Per marcare l'orario della decisione sulle pedine, si convalida quindi con <em>INVIO</em> nel momento in cui la mossa è finita nell'immagine. <em>v</em> fissa a posteriori l'istante corrente come istante della mossa del cursore, <em>MAIUSC-V</em> come istante del lancio. Un gesto di cubo convalida allo stesso modo, senza istante. Una mossa completata sulla plancia senza dadi digitati prima non ha istante del lancio. <em>SPAZIO</em> avvia o mette in pausa il video; <em>MAIUSC-SINISTRA</em> e <em>MAIUSC-DESTRA</em> lo spostano di 5 secondi, <em>CTRL-MAIUSC-SINISTRA</em> e <em>CTRL-MAIUSC-DESTRA</em> di un secondo.</p>
<p>Dai marcatori si deducono le <strong>durate</strong> delle decisioni: la decisione sulle pedine va dal lancio alla fine della mossa, la decisione sul videau dall'azione precedente al lancio, un raddoppio o una risposta dall'azione precedente alla propria. Una cella che porta un marcatore lo segnala con un punto discreto; il suo suggerimento indica il lancio, la mossa e la durata (“lancio 12:34, mossa 12:51, 17 s”), e il pannello di analisi mostra la durata dell'azione del cursore come per un match giocato in Duello. Un marcatore anteriore a quello che lo precede è segnalato come “marcatore a ritroso”, come ogni incoerenza, e lascia sconosciute le durate che ne dipendono. Posizionare il cursore su una cella (clic, <em>h</em>, <em>l</em>) porta il video un secondo prima del lancio di quell'azione, o prima della sua mossa se il lancio non ha istante; un lettore in pausa resta in pausa.</p>
<p>L'applicazione non decodifica il video da sola: i formati letti sono quelli della webview. Un formato che questa non legge viene segnalato nel riquadro, con il suo contenitore e, su Linux, i plugin GStreamer da installare: <code>gstreamer1.0-plugins-good</code> e <code>gstreamer1.0-libav</code> su Debian e Ubuntu, <code>gstreamer1-plugins-good</code> e <code>gstreamer1-plugin-libav</code> su Fedora, <code>gst-plugins-good</code> e <code>gst-libav</code> su Arch (vedere Download e installazione).</p>
<div class="admonition tip">
<p>Fare riferimento a Scorciatoie da tastiera per le scorciatoie disponibili.</p>
</div>
<h3>Pannello Tornei</h3>
<p>Il pannello <strong>Tornei</strong> (<em>CTRL-Y</em>) permette di raggruppare i match in tornei per un monitoraggio organizzato e un'analisi statistica per evento. I tornei possono essere creati, rinominati ed eliminati; i match possono essere assegnati ad essi. Le statistiche del pannello Stats possono essere filtrate per torneo. Premere <em>CTRL-Y</em> per mostrare o nascondere il pannello.</p>
<p><strong>Nuovo torneo</strong> apre il campo di creazione, che prende il focus; <em>ESC</em> o <strong>Annulla</strong> lo richiude. Un clic evidenzia una riga, un doppio clic o <em>INVIO</em> apre il torneo: le sue note, poi i suoi match, uno per riga, che un doppio clic apre, che ▲ e ▼ riordinano, che ⇄ scambia di giocatori e che × toglie dal torneo. Il campo <strong>Aggiungi un match…</strong> vi colloca un match del database, e ← riporta all'elenco dei tornei.</p>
<p>I tornei si riempiono da soli all'importazione. I file XG, GnuBG e BGF nominano il loro evento; quando un match nuovo viene importato, blunderDB lo classifica nel torneo con quel nome e lo crea se non esiste ancora. La data e il luogo del torneo restano vuoti: è qui che si compilano. Un match già presente nel database non viene mai riclassificato: reimportarne il file non disfa la sistemazione fatta a mano.</p>
<p>Le colonne <strong>PR</strong> e <strong>MWC</strong> di ogni torneo mostrano il PR e la perdita di MWC del <strong>giocatore di riferimento</strong> — vale a dire il giocatore presente nel maggior numero di match del torneo (in caso di parità, quello che ha preso più decisioni). Il PR non mescola quindi il vostro gioco con quello dei vostri avversari: per i vostri tornei, riflette la vostra prestazione da sola. Il nome del giocatore di riferimento compare in un suggerimento al passaggio sul valore.</p>
<h3>Dirigere un torneo</h3>
<p>blunderDB sa <strong>dirigere</strong> un torneo, non soltanto archiviarlo. La direzione è retta dal motore <strong>Nicomaque</strong>, di Nicolas Harmand: è lui che tiene il formato, gli abbinamenti, i tabelloni e la classifica; blunderDB gli dà la sua interfaccia e conserva i suoi match. Il pulsante <strong>ⓘ</strong> dell'intestazione della Direzione ricorda questo credito e porta al repository e alla documentazione del motore.</p>
<p>Un torneo diretto si sceglie nel pannello Pannello Tornei (<em>CTRL-Y</em>, comando <code>direct</code>): aprire un torneo, poi <strong>Dirigi questo torneo</strong>. Un torneo già diretto mostra il suo stato accanto al nome e il pulsante diventa <strong>Apri la direzione</strong>. Finché una direzione è aperta, l'area principale mostra il torneo <strong>al posto del tavoliere</strong> — è l'unica eccezione di blunderDB a questa regola; passare a qualsiasi altra scheda riporta il tavoliere. <strong>Esci dalla direzione</strong>, nell'intestazione della vista, o <strong>Chiudi la direzione</strong>, nel pannello, la chiudono.</p>
<p>Una direzione ha tre stati: <strong>in preparazione</strong> finché nessun match è stato lanciato, <strong>in corso</strong> poi, <strong>terminato</strong> una volta chiuso il torneo e congelata la classifica. Riaprire un torneo chiuso è possibile, e richiede una conferma: la classifica finale cessa di essere definitiva.</p>
<p>Tutto ciò che viene deciso è scritto in un <strong>registro</strong>, e nient'altro lo è. La classifica, i tabelloni, le proposte e gli avvisi sono riprodotti da questo registro a ogni apertura: un'interruzione di corrente non costa nulla, e una correzione non cancella mai ciò che è accaduto — si aggiunge.</p>
<h4>La pagina Direzione</h4>
<p>È qui che il direttore passa la maggior parte del tempo. Dall'alto in basso: gli <strong>avvisi</strong> del motore, che restano visibili e non bloccano mai nulla; la <strong>griglia dei tavoli</strong>, la cui intestazione porta il pulsante <strong>Stampa il foglio</strong> degli abbinamenti; l'<strong>ultima decisione</strong>; la <strong>coda delle proposte</strong>; e i giocatori liberi. La griglia precede la coda: una coda lunga non la spinge mai fuori dallo schermo. Con la tastiera, la prima sosta di <em>TAB</em> nella pagina è «Vai alla coda», che porta il fuoco sulla coda senza attraversare la griglia; la coda ricorda sotto il titolo le sue scorciatoie (<em>J</em> e <em>K</em>, <em>INVIO</em>, <em>SINISTRA</em> e <em>DESTRA</em>).</p>
<p>La vista occupa tutta la larghezza dell'area principale, e ogni scheda scorre per conto suo: uscire da una scheda e tornarci, o passare da una prova all'altra, ripristina la posizione in cui era stata lasciata. I pulsanti e i campi sono alti almeno 44 pixel, per essere colpiti senza precisione al banco; il numero di colonne della griglia segue la larghezza dell'area, non quella della finestra. Nelle <strong>Impostazioni</strong>, ogni sezione si ripiega sul proprio titolo, e il pulsante <strong>Apri nel browser</strong> delle Impostazioni e il pulsante <strong>Pagina murale</strong> dell'intestazione aprono la pagina murale con un clic non appena è scelta una cartella di uscita; la barra di stato indica il percorso del file scritto. Iscrizione, ritardatario, fase successiva, sorteggio, lancio di un match e chiusura vi lasciano anche un riscontro («Sophie Martin iscritta — 16 iscritti»).</p>
<p>Una Direzione si apre sui suoi <strong>Giocatori</strong> finché il torneo non è avviato, poi sulla scheda <strong>Direzione</strong>, e sulla scheda in cui era stata lasciata quando la si riapre. Dopo un ricaricamento o un riavvio dell'applicazione, l'ultima Direzione aperta si riapre da sola, sulla stessa scheda. Un secondo clic su <em>Dirigi</em> o su <em>Apri la direzione</em> mentre la vista si carica viene ignorato.</p>
<p>Un turno proposto si annuncia prima di avviarlo: <strong>Prossimo turno…</strong>, accanto a <em>Stampa il foglio</em>, chiede la data e l'ora da stampare («lunedì 21/09, ore 20») e stampa il foglio degli abbinamenti della coda, contrassegnato come «annunciato». Nulla viene avviato né scritto nel registro: il turno si avvia il giorno stabilito, alla sua ora. Un abbinamento in attesa di un tavolo libero porta un trattino al posto del numero di tavolo.</p>
<p>In cima alla vista del torneo, la <strong>barra dell'orologio</strong> sta in una riga: l'ora, il tempo trascorso dal primo match avviato, i match giocati e in corso, il ritmo osservato in minuti per punto rispetto a quello previsto, i match lenti, la prossima pausa e la <strong>fine stimata</strong>. La fine stimata è la previsione del motore: rigioca il registro, termina il torneo quindici volte al ritmo previsto, e la barra ne dà la mediana, spostata dopo le pause dichiarate. Una notte non dichiarata come pausa conta quindi come gioco. Un orario che non è di oggi porta il suo giorno.</p>
<p>Dal secondo giorno, il tempo trascorso lascia il posto al <strong>giorno di gioco</strong> (il giorno del primo match avviato è il giorno 1) e al <strong>tempo di gioco</strong>: il tempo durante il quale almeno un match era in corso, senza le notti né gli intervalli in cui nessun tavolo giocava. Un torneo chiuso non ha più la barra dell'orologio.</p>
<p>Una proposta si conferma con <strong>un clic</strong> su <em>Lancia</em>. <strong>Lancia tutto</strong> conferma in due clic le proposte che hanno un tavolo, dopo averne mostrato l'elenco; <em>Conferma</em> è in testa a quell'elenco. Gli abbinamenti senza tavolo restano in coda, segnati «nessun tavolo libero»: in modalità a turni, un turno resta aperto finché tutti i suoi giocatori non vi sono impegnati. Anche un ripescaggio tra pari merito attende la scelta del direttore, e con esso il passaggio di fase. «Ignora per ora» non scrive nulla: il motore è deterministico, e la proposta torna identica alla chiamata successiva. <em>Abbina a mano</em> resta sempre disponibile — il motore propone, il direttore decide.</p>
<p>Un match abbinato a mano senza numero di tavolo prende il primo tavolo libero; un tavolo dove è in corso una partita viene rifiutato. Se tutti i tavoli sono occupati, viene avviato comunque e la sua casella compare in fondo alla griglia, «senza tavolo», finché non lo si sposta su un tavolo.</p>
<p>Una proposta può portare un'osservazione del motore: nessun tavolo libero, o una fine partita prevista durante una pausa. Resta lanciabile in entrambi i casi. Quando una fase funziona a <strong>micro-turni</strong>, la coda mostra il tempo che manca al lotto successivo; alla scadenza le proposte compaiono da sole, e nulla si lancia da solo.</p>
<h4>La scheda del risultato</h4>
<p>Un clic su un tavolo occupato apre la scheda della partita. Mostra due grandi bersagli: i <strong>nomi dei due giocatori</strong>. Cliccare quello che ha vinto registra il risultato — due clic in tutto, tavolo compreso. Il vincitore è l'unica cosa richiesta; il punteggio è libero, l'uno, entrambi o nessuno. Da tastiera, <em>SINISTRA</em> o <em>DESTRA</em> sceglie il vincitore e <em>INVIO</em> lo registra. La scheda si chiude solo quando il risultato è scritto: un errore la lascia aperta, con il suo messaggio.</p>
<p>Il pulsante <strong>⋯</strong> della scheda apre ciò che serve di rado: il forfait — ogni pulsante nomina l'assente e chi vince —, una nota libera («caduto per tempo», «abbandonato per motivo di…»), lo spostamento della partita su un altro tavolo e il suo annullamento. Il forfait e l'annullamento si confermano; il forfait con un pulsante <strong>Dichiara forfait</strong>, che propone anche di ritirare il perdente nella stessa scheda. Spostata su un tavolo occupato, la partita scambia il suo tavolo con quella che lo occupa: due partite non condividono mai un tavolo, e lo stesso gesto le rimette a posto. Se un vecchio registro ne ha lasciate due su un tavolo, la griglia mostra le due caselle, segnalate, finché non se ne sposta una.</p>
<p>Un errore di digitazione visto subito si riprende in due clic sotto la griglia: <strong>Correggi</strong> l'ultima decisione, poi il vincitore giusto (<em>CTRL-Z</em> apre la stessa ripresa). Una correzione più vecchia si fa dalla cronologia.</p>
<h4>I menu contestuali</h4>
<p>Un clic destro, il tasto <em>MENU</em> o <em>MAIUSC-F10</em> su un oggetto della pagina Direzione apre le sue azioni abituali senza passare dalla scheda: una casella della griglia (libera od occupata), un giocatore (scheda <strong>Giocatori</strong>, giocatori liberi), un posto del tabellone, un posto, una proposta della coda, una riga della cronologia. Il menu si apre sull'oggetto; <em>SU</em> e <em>GIÙ</em> lo scorrono, <em>INVIO</em> sceglie, <em>ESC</em> lo chiude e restituisce il focus all'oggetto.</p>
<ul>
<li>Casella occupata: inserire il risultato, ritiro dell'uno o dell'altro, cambiare tavolo (puntare a un tavolo occupato scambia i due incontri), annullare l'incontro, cronologia di ciascun giocatore.</li>
<li>Casella libera: avviare qui l'abbinamento selezionato, mettere il tavolo fuori servizio o rimetterlo in servizio (tavoli di un evento). Un tavolo riservato a un'altra prova non propone nulla.</li>
<li>Giocatore: inserire il risultato del suo incontro in corso, andare al suo tavolo, cronologia, abbinare a mano con un altro giocatore libero, andare all'altra prova in cui gioca anch'egli, segnare assente o presente, ritirare ora o dopo il suo incontro, reiscrivere, correggere la scheda.</li>
<li>Posto senza incontro: collegare un incontro importato che questo posto sta aspettando.</li>
<li>Abbinamento: avviare, avviare a un tavolo…, cambiare la lunghezza…, abbinare diversamente (queste tre voci aprono l'abbinamento a mano con i due giocatori, la lunghezza e il tavolo dell'abbinamento), ignorare per ora, stampare il foglio del turno.</li>
<li>Riga della cronologia: correggere o annullare, aggiungere una nota, filtrare su uno dei giocatori.</li>
</ul>
<p>Il ritiro, l'annullamento di un incontro e il ritiro di un giocatore mantengono la conferma che hanno nella scheda e nei pulsanti delle righe. Mentre un'azione è in corso, le voci che agiscono sono disattivate, come i pulsanti. Un solo menu è aperto alla volta: aprirne un secondo chiude il primo.</p>
<p>Con la tastiera, la griglia occupa una sola sosta di <em>TAB</em>: ogni casella riceve il focus, anche quelle libere, e <em>SINISTRA</em>, <em>DESTRA</em>, <em>SU</em>, <em>GIÙ</em>, <em>HOME</em> e <em>FINE</em> passano da una casella all'altra. Una cifra apre la scheda del tavolo con quel numero; per un tavolo oltre il 9, la seconda cifra si digita entro 0,4 s. <em>M</em> apre la scheda sul campo del tavolo, e anche <em>X</em>, che la scheda sia già aperta o no: puntare a un tavolo occupato scambia i due incontri.</p>
<p>Con il mouse, si <strong>trascina</strong> una casella occupata su un'altra: su una casella libera l'incontro cambia tavolo; su una casella occupata, una riga «Tavolo 3 ↔ Tavolo 7?» chiede di confermare lo <strong>scambio</strong> dei due incontri. Un fantasma segue il puntatore e la casella di destinazione viene evidenziata; <em>ESC</em> annulla il gesto, e non viene scritto nulla finché il puntatore non è rilasciato su una casella. Un tavolo fuori servizio viene rifiutato e la barra di stato ne dà il motivo. Da riga di comando, <code>blunderdb tournament move</code> esegue lo stesso spostamento o scambio (vedi Interfaccia a riga di comando (CLI)).</p>
<h4>Schermo intero</h4>
<p>Il tasto <em>F11</em> della pagina Direzione, o il pulsante in basso a destra, la mette a schermo intero: la barra degli strumenti, le schede, la scacchiera, il pannello e la barra di stato scompaiono e l'intera finestra passa alla direzione. La modalità attraversa le schede della direzione (Direzione, Giocatori, Cronologia, …). Un menu o una scheda aperti si chiudono prima con <em>ESC</em>; un secondo <em>ESC</em>, o <em>F11</em>, esce dallo schermo intero e riporta la finestra allo stato precedente. Anche lasciare la pagina, cambiando scheda dell'applicazione o chiudendo la direzione, vi pone fine.</p>
<h4>La ricerca rapida</h4>
<p>Il tasto <em>/</em> della pagina Direzione apre la tavolozza sul solo torneo: giocatori, tavoli, match in corso e prove dell'evento aperto, o della prova da sola quando non è in nessun evento. Si digita un nome, un club o un numero di tavolo («4» o «t4»); i giocatori a un tavolo vengono prima dei giocatori liberi. <em>INVIO</em> porta all'oggetto: un giocatore in un match, un match o un tavolo occupato aprono la scheda del tavolo, un tavolo libero riceve il focus nella griglia, un giocatore libero compare nella scheda <strong>Giocatori</strong> filtrato sul suo nome, una prova diventa la scheda corrente. Un risultato di un'altra prova dell'evento cambia prima prova. <em>ESC</em> chiude senza aprire nulla, e in un campo di testo il tasto resta una barra. <em>CTRL-MAIUSC-P</em> apre la tavolozza completa, che contiene anche il torneo.</p>
<h4>I giocatori</h4>
<p>La scheda <strong>Giocatori</strong> iscrive, corregge e ritira. Il campo d'iscrizione mantiene il focus e si svuota dopo ogni nome: venti giocatori si iscrivono con la sola tastiera. Il completamento automatico propone i giocatori della base; sceglierne uno fissa l'ortografia esatta che portano le sue partite e precompila il suo punteggio con il PR.</p>
<p>La <strong>rubrica</strong> raccoglie gli iscritti di tutti i tornei diretti della base, senza duplicati per nome, con il club e il punteggio della loro ultima iscrizione. Non è mai memorizzata: eliminare una direzione ne toglie gli iscritti. Riprendere gli iscritti di un torneo precedente è un clic, quanti che siano; la rubrica si copia in CSV o si salva in un file (<strong>Salva…</strong>), e si rilegge incollata.</p>
<p>Un torneo di doppio iscrive coppie: la casella <strong>Coppia</strong> aggiunge nome, club e valutazione del compagno. La coppia gioca con il nome «A / B», che portano anche le sue partite; la sua valutazione è la media delle due, e un valore inserito in <strong>Valutazione della coppia</strong> la sostituisce. L'annuario conserva le due persone, mai la coppia.</p>
<p>Prima di <strong>Iscrivi</strong>, l'anteprima di un CSV incollato elenca le righe illeggibili — senza nome, senza separatore quando le altre righe ne hanno uno, un punteggio che non è un numero — e i doppioni, all'interno dell'incollato o con un giocatore già iscritto. Un doppione non viene iscritto, a meno di spuntarne la casella.</p>
<p>Il campo filtro dell'elenco conserva il suo testo quando si cambia prova: finché è attivo, accanto compare un'etichetta <strong>filtro: …</strong> e la sua croce la cancella.</p>
<p>Un <strong>ritardatario</strong> arrivato dopo il sorteggio prende un bye libero se il tabellone ne offre uno, e l'interfaccia scrive accanto al campo dove entrerà prima di convalidare. Senza posto libero, viene iscritto lo stesso e la vista dice in quale fase entrerà. Nessun sorteggio già fatto viene rifatto.</p>
<p>Un ritiro avviene <em>ora</em> oppure <em>dopo la partita in corso</em>, a seconda che il giocatore vada via subito o finisca quello che sta giocando; si conferma con un pulsante <strong>Ritira</strong>, non rosso perché nulla viene eliminato.</p>
<p>Un giocatore che salta un turno non ha bisogno di essere ritirato: <strong>Segna assente</strong>, sulla sua riga, apre un piccolo modulo sotto il suo nome — <em>fino a</em> un'ora (precompilata con l'ora successiva), oppure, quando la fase in corso è uno svizzero a turni, <em>fino al turno</em> con il suo numero. Il motore smette allora semplicemente di abbinarlo, ma il suo rango, le sue vite e il suo posto nel tabellone restano quelli che si è guadagnato — l'assenza non è un forfait. <strong>Rientra</strong>, sulla sua riga, toglie l'assenza con un clic, prima o dopo la scadenza dichiarata.</p>
<p>Correggere la scheda di un giocatore ritirato — nome, circolo, punteggio — lo lascia ritirato. Il suo ritorno è un gesto a parte: <strong>Reiscrivi</strong>, sulla sua riga. Torna a essere abbinato, con i risultati e le vite che aveva quando si è ritirato; i match persi a tavolino al ritiro restano persi.</p>
<p>Un qualificato di girone che si ritira prima del sorteggio della fase successiva lascia un posto libero. La coda propone allora un <strong>ripescaggio</strong>: il successivo del suo girone, né qualificato né ritirato, con il maggior numero di vittorie di girone, prende il suo posto. Tra pari merito, la coda propone ciascuno di essi e il direttore sceglie; «Fase successiva» senza ripescaggio lascia il posto in esenzione. Una volta sorteggiato il tabellone, il ritirato perde il suo match per forfait.</p>
<h4>Tabelloni, posti, classifica, cronologia</h4>
<p>La scheda <strong>Tabelloni</strong> disegna i tabelloni con le loro linee di collegamento, dal primo turno alla finale, la consolazione accanto al tabellone principale e, per uno svizzero, la tabella delle vite. Un girone si legge come tabella incrociata dei risultati. Un tabellone non ancora sorteggiato mostra il suo scheletro in grigio. Una partita già giocata porta il suo risultato; una partita segnalata dal motore è contrassegnata sul posto. Un punto accanto al nome della scheda indica che un tabellone è in corso.</p>
<p><strong>Cliccare un posto</strong> (o premere Invio su un posto focalizzato) apre su di esso la stessa scheda della griglia dei tavoli: il vincitore di una partita in corso si inserisce con due clic, e una partita terminata si corregge cliccando il nome del vero vincitore.</p>
<p>La scheda <strong>Match</strong> (intitolata «Posti») collega il torneo alla libreria. Ogni match del torneo è un posto, che si riempie in due modi: trascrivere il match subito (Pannello Trascrizione), oppure agganciarvi un match già importato. <strong>Nulla viene agganciato per deduzione</strong>: una coincidenza di nomi è un suggerimento da accettare, un abbinamento parziale non è nemmeno suggerito, e se il file di un match agganciato contraddice il risultato registrato, lo scarto è mostrato senza essere risolto — durante un torneo, la parola del direttore fa fede.</p>
<p>La scheda <strong>Classifica</strong> mostra la classifica corrente, sezione per sezione, con il bilancio di ciascuno (vittorie–sconfitte) e i premi quando è impostato un montepremi. Due pari merito condividono il posto e il premio. Un giocatore ritirato conserva il posto che il suo percorso gli vale, segnato «ritirato» con il suo bilancio o il punto del tabellone in cui si è fermato. <strong>Chiudi il torneo</strong> congela la classifica finale. La classifica si copia in CSV, nella lingua dell'interfaccia, con la sezione e l'ultima fase in cui ciascuno è entrato, o si salva in un file: <strong>Salva…</strong> apre la finestra di dialogo del sistema su un nome proposto, il torneo seguito dalla parola «classifica» e dalla data odierna, e il file contiene esattamente il CSV copiato. Da riga di comando, <code>blunderdb tournament standings</code> scrive lo stesso CSV. Il nome di un giocatore è un link: apre la Cronologia filtrata su di lui, dove ciascuno dei suoi risultati si corregge.</p>
<p>Chiudere senza match in corso è un clic; quando tutto è giocato, la coda propone anche <strong>Chiudi il torneo</strong>, che si lancia come un'altra proposta. Con match in corso, la Classifica dice quanti sono e attende un secondo clic sul posto: chiudere congela la classifica senza di loro, e il loro risultato non si inserisce più. <strong>Riapri</strong> si conferma allo stesso modo; la classifica finale smette allora di essere definitiva, e la riapertura resta nel registro.</p>
<p>La scheda <strong>Cronologia</strong> è il registro in chiaro: una riga per decisione, in ordine, filtrabile per giocatore o per partita. È ciò che un direttore rilegge dopo una contestazione, ed è lì che una decisione più vecchia si corregge o si annota. Una riga di risultato nomina i due giocatori: il vincitore e il suo avversario.</p>
<p><strong>Correggi</strong>, sulla riga di un risultato, apre sotto di essa la stessa ripresa dell'ultima decisione: si clicca il vincitore giusto, con il punteggio se serve. Il risultato originale resta al suo posto nel registro, la correzione vi si aggiunge, e la classifica ne tiene conto subito.</p>
<h4>Le impostazioni</h4>
<p>La scheda <strong>Impostazioni</strong> si apre su <strong>formati con nome</strong>: sei tornei di club pronti all'uso, il primo dei quali è consigliato. Sceglierne uno basta per iniziare; i campi restano modificabili in seguito.</p>
<p>Si impostano qui: le fasi (<strong>Aggiungi una fase</strong> la aggiunge dopo le altre; una fase già aperta non si toglie) e la lunghezza dei loro match, quella della finale, le lunghezze turno per turno di un tabellone («15, 13, 11» si legge dall'ultimo turno all'indietro), la <strong>lunghezza finale</strong> (match più lunghi a partire da un numero di giocatori ancora in gara), i <strong>micro-turni</strong> (abbinare a lotti ogni N minuti), il numero di tavoli, il ritmo previsto in minuti per punto (8 per impostazione predefinita; la barra dell'orologio e la fine stimata ne partono), le pause della giornata, il montepremi (quota d'iscrizione, trattenuta del club, tabella per sezione) e la cartella di visualizzazione.</p>
<p>Un <strong>tabellone</strong> ha tre caselle: <strong>Consolazione</strong> (i suoi sconfitti giocano un secondo tabellone, una sezione a parte in classifica), <strong>Riconciliazione</strong> (il vincitore della consolazione affronta quello del tabellone principale; la casella compare solo con la consolazione) e <strong>Ricarica</strong> (in doppia eliminazione, il vincitore del tabellone principale deve essere battuto due volte; la casella compare solo con la riconciliazione). Una consolazione ha una classifica propria solo con una scala di premi: finché quella della sezione <em>Consolazione</em> è vuota, le Impostazioni lo ricordano. Una fase a <strong>Gironi</strong> si imposta con la dimensione dei gironi (4 per impostazione predefinita) e il numero di qualificati per girone (2 per impostazione predefinita).</p>
<p>La <strong>soglia</strong> di una fase svizzera è la somma delle vite rimaste a partire dalla quale si passa al tabellone. Se la somma delle vite iniziali (giocatori × vite) è già inferiore o uguale alla soglia, la fase svizzera verrebbe saltata: le Impostazioni lo segnalano prima del primo avvio.</p>
<p>Le impostazioni restano accessibili <strong>durante il torneo</strong>: abbassare la soglia alle 22 per finire prima, aggiungere una consolazione il sabato sera, finché il tabellone non è sorteggiato. Ciò che è allora bloccato appare in grigio con il suo motivo: il formato di una fase aperta, il numero di vite che ha distribuito e, dal sorteggio di una fase, la dimensione dei suoi gironi e il numero di qualificati. Salvare durante il torneo mostra prima l'elenco di ciò che cambierà e chiede conferma. Ciò che il motore rifiuta vi figura con il suo motivo, e allora nulla viene salvato: è il caso della consolazione, della riconciliazione o della ricarica di un tabellone già sorteggiato.</p>
<p>Un tavolo con la tavola rotta si dichiara in <strong>Tavoli fuori servizio</strong>: i suoi numeri, separati da virgole («7, 12»). Il motore non lo assegna più, e la griglia lo mostra non disponibile; abbassare il numero di tavoli toglierebbe l'ultimo, non quello rotto. Se un match è in corso su un tavolo messo fuori servizio, l'elenco di ciò che cambierà lo dice e indica un tavolo libero dove spostarlo, dalla scheda del match.</p>
<p>Le <strong>teste di serie</strong> sono un'opzione, disattivata per impostazione predefinita: lo studio del motore conclude «nessuna testa di serie protetta», che è la cultura attuale del backgammon. Attivate, i giocatori sono piazzati per punteggio.</p>
<h4>Le prove di un evento</h4>
<p>Più prove giocate sugli stessi tavoli — una principale, una speed, un doppio — si raggruppano in un <strong>Evento</strong>, in cima alle Impostazioni (ripiegato sul suo titolo quando la prova vi è collegata; il pulsante <strong>Impostazioni dell'evento</strong> della scheda <em>Tutti i tavoli</em> vi conduce): <strong>Crea e collega…</strong> crea l'evento con il suo numero di tavoli, <strong>Collega…</strong> vi aggiunge una prova diretta. Collegare mostra prima cosa cambierà: i tavoli della prova diventano quelli dell'evento. <strong>Stacca dall'evento</strong> restituisce la prova a sé stessa, con il suo registro e i suoi tavoli; <strong>Elimina l'evento</strong>, dopo conferma, lo mette nel cestino e stacca le sue prove senza eliminarne alcuna.</p>
<p>In un evento nessuna prova propone un tavolo dove ne gioca un'altra: la griglia mostra quei tavoli occupati, con il nome della prova, e un abbinamento senza tavolo libero attende. Un tavolo fuori servizio si spunta una volta, nell'evento, e vale per tutte le sue prove; il numero di tavoli, i tavoli fuori servizio e le pause modificati nelle Impostazioni di una prova valgono anche per l'evento, e l'elenco di cosa cambierà nomina le altre prove.</p>
<p>Le <strong>proprietà dei tavoli</strong> si impostano nel pannello Evento: una tabella con una riga per tavolo, con il suo <strong>nome</strong> («Stream», per esempio), la sua <strong>sala</strong> (un'etichetta libera), una casella <strong>Riservato</strong> e i giocatori a cui è <strong>Assegnato a</strong>, scelti tra gli iscritti alle prove dell'evento. Per quaranta tavoli, <em>Tavoli da N a M, sala</em> imposta una sala su un intero intervallo in un colpo solo; <strong>Salva</strong> scrive solo i tavoli che hanno una proprietà, gli altri restano tavoli ordinari. Un tavolo riservato non viene mai proposto, ma vi si può collocare un incontro a mano, con un lancio, uno spostamento o un trascinamento. Un tavolo assegnato riceve per primo l'incontro del suo titolare, quando è libero; altrimenti l'incontro riceve un tavolo ordinario, e fuori dagli incontri dei suoi titolari si comporta come un tavolo riservato. Due titolari di tavoli diversi che si incontrano giocano sul più piccolo dei due.</p>
<p>Una <strong>sala</strong> è l'insieme dei tavoli che portano la stessa etichetta: «sala A» per i tavoli da 1 a 20, «sala B» per i successivi. Nelle Impostazioni di ciascuna prova collegata, <em>Sale in cui gioca questa prova</em> spunta le sale in cui gioca: il DMP in B, lo speed in A. Senza alcuna casella spuntata, sono tutti i tavoli; una prova non riceve alcuna proposta fuori dalle sue sale e non può spostarvi un incontro. Togliere una sala che ospita un incontro in corso della prova viene rifiutato, indicando il tavolo. Una prova che gioca da sola imposta le stesse proprietà dei tavoli nelle proprie Impostazioni, senza sale di prova.</p>
<p>La <strong>Classifica di stagione</strong>, nel pannello Evento, somma le prove chiuse dell'evento: ogni posto porta i punti del <strong>Punteggio</strong> (il vincitore per primo, 25, 18, 15, 12, 10, 8, 6, 4, 2, 1 per impostazione predefinita), e i pari merito si dividono la media dei posti che occupano. Una persona è riconosciuta da una prova all'altra dal nome. <strong>Elo di club</strong> aggiunge una colonna: ciascuno parte da 1500 e le partite della stagione sono rigiocate in ordine, secondo la formula di FIBS. <strong>Calcola</strong> mostra la classifica, <strong>Copia come CSV</strong> la copia con una colonna di punti per prova. Una prova non chiusa non porta nulla.</p>
<p>Aprire la Direzione di una prova di un evento apre anche le altre: in alto nella Direzione appare una scheda per prova, ciascuna con il proprio riepilogo — proposte in attesa, match in corso, un avviso se ce n'è uno. Cambiare prova è un clic sulla sua scheda, senza conferma; la prova lasciata non si chiude e non riesegue nulla, resta come la si è lasciata. Un torneo fuori da qualsiasi evento ha una sola prova: nessuna scheda da mostrare. Cambiare scheda dell'applicazione e poi tornare a Tornei restituisce la Direzione come la si era lasciata: la stessa prova o <em>Tutti i tavoli</em>, e la stessa scheda della vista.</p>
<p>La scheda <strong>Tutti i tavoli</strong>, a sinistra delle prove, mostra tutti i tavoli dell'evento in un'unica griglia: una casella per tavolo, qualunque sia la prova che lo occupa, contrassegnata dal nome e dal colore della sua prova. La scheda del risultato, i menu contestuali, la tastiera e il trascinamento funzionano come nella griglia di una prova, e ogni gesto si rivolge alla prova della sua casella. Trascinare un match su un tavolo occupato da un'altra prova scambia i due match dopo una conferma che nomina entrambe le prove. Sotto la griglia, le proposte di tutte le prove formano un'unica coda, ciascuna contrassegnata dalla sua prova e con il proprio pulsante <em>Avvia</em>: i giocatori che aspettano da più tempo passano per primi, così che una prova non aspetti la fine del turno di un'altra. Un match senza tavolo — tutti i tavoli occupati, o un giocatore impegnato in un match di un'altra prova — segue la coda con il suo motivo, senza pulsante <em>Avvia</em>, finché un tavolo o il giocatore non si libera. Fare clic su una prova o su una scheda di vista esce da <em>Tutti i tavoli</em>.</p>
<p>Quando l'evento ha più sale, la griglia è raggruppata per sala, sotto il nome di ciascuna. Il nome di un tavolo è mostrato accanto al suo numero, con una bandiera per un tavolo riservato e una stella seguita dai titolari per un tavolo assegnato; anche le proposte nominano il tavolo.</p>
<p>Una stessa persona può giocare più prove dell'evento: due Partecipanti con lo stesso nome sono la stessa persona, e per una coppia di doppio conta ciascuno dei due membri. Finché gioca in una prova, le altre non la propongono, e la loro lista <em>In attesa</em> dice dove gioca: «gioca nel principale, tavolo 4». Un match di tabellone che la aspetta resta nella coda, e avviarlo viene rifiutato finché gioca. L'abbinamento a mano resta consentito: il match inizia, e la sua casella della griglia porta la stessa indicazione.</p>
<h4>Visualizzazione del torneo</h4>
<p>Un torneo si guarda. Scegliere una <strong>cartella del display</strong> nelle Impostazioni basta una volta per tutte: blunderDB vi riscrive una pagina HTML autonoma a ogni cambiamento, e la pagina si ricarica da sola. Si apre offline, su un secondo schermo o proiettata, e non carica alcuna risorsa esterna. <em>Apri nel browser</em> la mostra subito.</p>
<p>Il <strong>foglio degli abbinamenti</strong> si posa sul tavolo dell'accoglienza: un clic su <em>Stampa il foglio</em> apre la finestra di stampa del sistema. Una riga per partita — i due giocatori, la lunghezza, il tavolo, due caselle vuote per il punteggio — e un turno di trentadue giocatori sta su una pagina A4.</p>
<p>Un evento ha una propria cartella di uscita, scelta una volta nel suo pannello Impostazioni con lo stesso pulsante <em>Scegli cartella</em>: blunderDB vi scrive <code>index.html</code>, la <strong>pagina murale</strong> dell'evento — una riga per tavolo, qualunque sia la prova che lo occupa, con i turni annunciati di ciascuna prova e un link alla sua pagina — e ogni prova collegata scrive la propria in una sottocartella. Un gesto in qualsiasi prova dell'evento rigenera la pagina murale; la cartella propria di una prova collegata è conservata ma ignorata finché resta nell'evento.</p>
<p>Quando una prova è in fase di tabellone e questo è stato sorteggiato, la sua pagina murale e quella dell'evento mostrano l'albero in grande, leggibile da lontano: una colonna per turno, con i perdenti che scendono nella consolazione tratteggiati. La pagina scorre da sola, senza script, tra il suo contenuto abituale (i tavoli, per l'evento) e l'albero di ogni prova a tabellone, dodici secondi ciascuno; si ricarica sempre ogni trenta secondi e riprende la rotazione da dove era. Una prova senza tabellone — una fase svizzera, ad esempio — non ha albero e la pagina resta quella di prima.</p>
<p>Sotto la voce «Gioco io?», la pagina di una gara e la pagina murale dell'evento nominano i giocatori che in questo momento non giocano, così come il motore li colloca: «eliminato/a» quando non resta più alcun match da giocare, «qualificato/a» con il nome della fase in cui entra il giocatore, «non ancora deciso» quando la sua sorte dipende dalla fine della fase o da un ripescaggio che il direttore non ha risolto, «esentato/a — entra al turno N» per un giocatore sorteggiato senza avversario nel tabellone, finché non ha giocato, «esentato/a — rigioca al turno N» per l'esentato di un turno svizzero, e «vincitore/vincitrice». La classifica subentra una volta terminato il torneo.</p>
<p>Fuori dall'interfaccia, il sottocomando <code>blunderdb tournament</code> rilegge un torneo diretto senza interfaccia grafica: <code>list</code>, <code>verify</code>, <code>standings</code>, <code>page</code> ed <code>export</code>; <code>proposals</code> mostra la coda del motore, numerata, e <code>confirm</code> ne conferma una, ripescaggio compreso; <code>ranking --season</code> cumula i tornei chiusi di un evento o di un periodo in una classifica di stagione, con una tabella di punti per posto e, a scelta, un Elo di club (due iscritti con lo stesso nome in una stessa prova chiusa fanno rifiutare la classifica, che conosce una persona dal suo nome); <code>page --rencontre</code> scrive la pagina murale di un evento anziché la pagina di una sola prova; <code>hall</code> stampa la griglia <em>Tutti i tavoli</em> di un evento e la sua coda, <code>tables</code> le proprietà dei tavoli e le sale delle prove. Vedi Interfaccia a riga di comando (CLI).</p>
<h3>Pannello Stats</h3>
<h4>Introduzione</h4>
<p>Il pannello <strong>Stats</strong> permette di analizzare il proprio livello di gioco e di seguire la propria progressione nel tempo a partire dalle posizioni importate nel database. Calcola e visualizza gli indicatori <strong>PR</strong> (<em>Performance Rating</em>) e <strong>MWC cost</strong> (Match Winning Chance cost) per l'insieme delle posizioni o per un sottoinsieme filtrato.</p>
<p>Il pannello Stats è particolarmente utile per:</p>
<ul>
<li><strong>collocare il proprio livello</strong> rispetto alle fasce di livello (<em>Classe mondiale</em>, <em>Esperto</em>, <em>Avanzato</em>…) grazie al PR globale;</li>
<li><strong>seguire la propria progressione</strong> torneo dopo torneo o match dopo match grazie ai grafici della scheda Progressione;</li>
<li><strong>individuare i propri punti deboli</strong>: la scheda Errori mostra la ripartizione tra mosse di pedine e decisioni di cubo e la distribuzione delle magnitudo d'errore;</li>
<li><strong>confrontare fra loro i giocatori del database</strong>, una riga per giocatore, grazie alla scheda Giocatori — utile per seguire un'intera competizione;</li>
<li><strong>accedere direttamente alle posizioni interessate</strong> cliccando su qualsiasi indicatore (drill-down).</li>
</ul>
<h4>Apertura del pannello</h4>
<p>Per aprire il pannello Stats:</p>
<ul>
<li>Premere <em>CTRL-D</em>.</li>
<li>Digitare il comando <code>stats</code> o <code>st</code> nella riga di comando.</li>
</ul>
<div class="admonition note">
<p>Il pannello si aggiorna automaticamente a ogni modifica del filtro. Non ricalcola le statistiche in caso di semplice passaggio PR ↔ MWC: entrambe le metriche vengono calcolate simultaneamente dal backend.</p>
</div>
<h4>Barra dei filtri</h4>
<p>La barra dei filtri, in alto nel pannello, permette di limitare il calcolo a un sottoinsieme di posizioni.</p>
<h5>Prospettiva del giocatore</h5>
<p>Il menu a discesa <strong>Giocatore</strong> permette di filtrare le statistiche in base al giocatore analizzato. blunderDB seleziona automaticamente il giocatore il cui nome compare più spesso nel database — modificabile in qualsiasi momento.</p>
<div class="admonition tip">
<p>Cambiare giocatore non comporta alcuna perdita di dati; è sufficiente riselezionare il giocatore precedente dall'elenco.</p>
</div>
<h5>Filtri disponibili</h5>
<ul>
<li><strong>Torneo(i)</strong> — restrizione a uno o più tornei. È possibile selezionare più tornei contemporaneamente.</li>
<li><strong>Date</strong> — intervallo temporale (<em>Da</em> … <em>A</em>). Se viene indicata solo la data di inizio, vengono incluse le posizioni più recenti.</li>
<li><strong>Tipo di decisione</strong> — Tutti / Mosse di pedine / Decisioni di cubo.</li>
<li><strong>Lunghezza del match</strong> — restrizione a lunghezze di match specifiche (1, 3, 5, 7, 9, 11, 13, 15, 21 punti). È possibile combinare più lunghezze.</li>
<li><strong>Motore</strong> e <strong>Profondità min.</strong> — conservare solo le decisioni analizzate da un dato motore (gnubg, xg…), o ad almeno una data profondità, in ply.</li>
</ul>
<p>Un pulsante <strong>↺ Reimposta</strong> azzera tutti i filtri (tranne il giocatore rilevato automaticamente).</p>
<div class="admonition note">
<p>I filtri vengono salvati nella configurazione di blunderDB (<code>config.yaml</code>) e ripristinati all'avvio successivo.</p>
</div>
<h4>Commutazione PR / MWC</h4>
<p>Il pulsante <strong>PR / MWC</strong> in alto nel pannello commuta la metrica visualizzata in tutte le schede.</p>
<p><strong>PR (Performance Rating)</strong></p>
<blockquote>
<p>L'errore medio di equità per decisione conteggiata, moltiplicato per 500 come fanno eXtreme Gammon e GNUbg: un PR di 5,0 equivale a 0,010 di equità persa per decisione, ossia 10 millesimi di punto (mpt). La regola esatta di conteggio — quali decisioni entrano nel denominatore, come il punteggio viene convertito — è quella di Appendice: modello statistico — allineamento XG / gnuBG / blunderDB.</p>
<p>Le fasce di livello che il pannello disegna dietro la curva di progressione sono un <strong>riferimento indicativo proprio di blunderDB</strong>: nessuna pubblicazione fa autorità su queste soglie. Il limite superiore di ogni fascia è escluso: un PR di 4 è <em>Avanzato</em>, non <em>Esperto</em>.</p>
<table>
<thead>
<tr>
<th>Livello</th>
<th>PR</th>
</tr>
</thead>
<tbody>
<tr>
<td>Classe mondiale</td>
<td>&lt; 2</td>
</tr>
<tr>
<td>Esperto</td>
<td>2 – 4</td>
</tr>
<tr>
<td>Avanzato</td>
<td>4 – 6</td>
</tr>
<tr>
<td>Intermedio</td>
<td>6 – 9</td>
</tr>
<tr>
<td>Occasionale</td>
<td>9 – 12</td>
</tr>
<tr>
<td>Principiante</td>
<td>≥ 12</td>
</tr>
</tbody>
</table>
</blockquote>
<p><strong>MWC cost (Match Winning Chance cost)</strong></p>
<blockquote>
<p>Probabilità cumulata di vittoria del match persa a causa degli errori, sull'insieme dei dati filtrato. Calcolata a partire dalla MET corrente del database, per impostazione predefinita la Kazaross-XG2 integrata in blunderDB. Un'analisi al punteggio di match calcolata con un'altra tabella è «MET diversa» e resta fuori dalle statistiche.</p>
<div class="admonition caution">
<p>Il MWC cost <strong>non è applicabile</strong> alle posizioni <em>money-game</em> (senza posta di match). Queste posizioni sono escluse dal calcolo MWC. I valori MWC dipendono dalla MET utilizzata; non sono direttamente confrontabili tra software che usano MET diverse.</p>
</div>
</blockquote>
<p><strong>Perdita di MWC (eq. 7 punti)</strong></p>
<blockquote>
<p>La probabilità di vincere il match che un giocatore ha perso sull'insieme delle sue decisioni — pedine, cubo, accetto o rifiuto, con cubo — riportata a un match ai 7 punti. Per un match ai <em>N</em> punti in cui il giocatore ha perso <em>L</em> di MWC:</p>
<blockquote>
<p>L₇ = L × √(7 / N)</p>
</blockquote>
<p>Per un match ai 7 punti, è la perdita di MWC del match, quella che mostra eXtreme Gammon. La radice viene dal modello della formula FIBS: a parità di forza, la perdita di un giocatore cresce come la radice della lunghezza del match. 7 punti è la lunghezza di riferimento perché è la più comune nei tornei.</p>
<p>Si legge come la quota di match persa contro un giocatore perfetto: 12,3 % significa che, invece del 50 % di probabilità contro il motore, il giocatore ne aveva solo il 37,7 % in un match ai 7 punti. Il suggerimento a comparsa ne dà la lettura in Elo contro il motore, invertendo la formula FIBS: D = (2000 / √7) × log₁₀(q / (1 − q)), con q = 0,5 − L₇. Oltre una perdita del 49 %, la formula non ha più un valore finito: l'Elo mostrato è allora un massimo («≤»). Le due cifre ordinano i giocatori allo stesso modo.</p>
<p>Completa il PR senza sostituirlo: il PR divide gli errori per un numero di decisioni, e questo numero dipende da ciò che si conta come decisione (mosse forzate, decisioni di cubo ovvie). La perdita di MWC non conta alcuna decisione: ogni errore pesa quanto è costato al punteggio in cui è stato commesso.</p>
<p>Su più match (statistiche di un giocatore o di un torneo), le perdite e le radici delle lunghezze si sommano prima del rapporto: L₇ = √7 × ΣL / Σ√N. Un match ai 7 punti pesa quindi più di un match a 1 punto, e un solo match dà il proprio valore.</p>
<p>Una partita isolata è molto rumorosa: bastano pochi errori gravi per raddoppiarne il valore. Il badge e il bilancio della partita (bilancio della partita) danno l'intervallo al 95 % ricampionando le sue partite; un aggregato (statistiche di un giocatore o di un torneo) lo dà ricampionando i suoi match. Servono almeno due partite o due match. Una riga per match delle statistiche non ha intervallo: filtrata per giocatore contiene una sola unità; senza giocatore, i suoi due lati misurerebbero lo scarto tra gli avversari, non l'incertezza. Non classificate i giocatori su un solo match.</p>
<div class="admonition caution">
<p>Una partita <em>money-game</em> non ha una lunghezza di match: la perdita di MWC (eq. 7 punti) non vi è definita e il pannello lo indica invece di mostrare un numero. Come il costo MWC, dipende dalla MET.</p>
</div>
<p>La terza scelta del pulsante, <strong>MWC 7 pts</strong>, traccia questa perdita nella scheda Progressione, e la mostra sulle schede della dashboard e nel confronto pedine/cubo della scheda Errori, suddivisa in pedine e cubo sugli stessi match (le due parti si sommano). La classifica dei giocatori ha la sua colonna. I grafici senza equivalente a 7 punti (per azione di cubo) conservano il MWC cost.</p>
</blockquote>
<p>La commutazione PR ↔ MWC è istantanea: non viene eseguito alcun ricalcolo da parte del backend.</p>
<h4>Il rapporto HTML</h4>
<p>Il pulsante <strong>Rapporto HTML</strong> nell'intestazione del pannello produce un documento <strong>autonomo</strong>: un solo file, senza immagini esterne, senza foglio di stile remoto, senza script. I diagrammi sono SVG in linea, disegnati dallo stesso rendering della dama a schermo, con la vostra tavolozza. Si apre in qualunque browser, viaggia per posta elettronica, e <strong>si stampa in PDF dal browser stesso</strong> — il che evita di imbarcare un generatore di PDF per produrre ciò che tutti hanno già.</p>
<p>Contiene gli indicatori del perimetro corrente (posizioni, incontri, decisioni contate, PR globale, pedine e cubo), poi le <strong>dieci decisioni più costose</strong>, ciascuna con il suo diagramma, il suo costo, l'incontro da cui viene e la mossa migliore quando un'analisi la fornisce.</p>
<p>Il documento è costruito dal motore, non dallo schermo: la riga di comando (<code>stats report --html</code>, vedi stats — Errori ricorrenti) e il demone HTTP (rotta <code>stats.report</code>) producono lo stesso rapporto, in una delle nove lingue dell'interfaccia. Solo l'applicazione grafica disegna i diagrammi con la tavolozza della scacchiera; le altre due usano la tavolozza predefinita.</p>
<p>Il rapporto porta il <strong>filtro corrente</strong> del pannello Stats. Un rapporto che non dichiara il proprio perimetro è un rapporto le cui cifre non vogliono dire nulla: regolate il filtro — un torneo, un intervallo di date, un giocatore — prima di produrlo.</p>
<h4>Scheda Cruscotto</h4>
<p>La scheda <strong>Cruscotto</strong> offre una visione sintetica degli indicatori chiave.</p>
<h5>Schede di livello</h5>
<p>Tre schede visualizzano il PR (o MWC) per:</p>
<ul>
<li><strong>PR Globale</strong> — tutte le decisioni (pedine + cubo);</li>
<li><strong>PR pedine</strong> — solo decisioni sulle pedine;</li>
<li><strong>PR Cubo</strong> — solo decisioni di cubo.</li>
</ul>
<p>Cliccando su una scheda si caricano nel pannello di analisi le posizioni del sottoinsieme corrispondente (drill-down).</p>
<div class="admonition note">
<p>Il numero totale di decisioni viene visualizzato in fondo a ciascuna scheda al passaggio del mouse.</p>
</div>
<h5>Piano di studio</h5>
<p>La scheda <strong>Piano di studio</strong> risponde a «che cosa devo lavorare adesso?». Gli errori ricorrenti (scheda Errori) dicono dove il filtro ha perso di più; non dicono dove lo studio rende di più. Una famiglia di posizioni davvero difficili costa cara a tutti, e tre errori non fanno una tendenza. Il piano corregge entrambe le cose.</p>
<ul>
<li>Una <strong>famiglia</strong> è un gruppo di errori ricorrenti: un piano di gioco, una natura di decisione (pedine o cubo) e un tema. Gli errori senza tema non ne formano: non indicano nulla da studiare.</li>
<li>Ogni errore è quantificato in <strong>MWC</strong>, come nel pannello Match: la sua perdita ℓ e la sua <em>difficoltà</em> d, la perdita che un giocatore di riferimento avrebbe subito nella stessa posizione (vedere la difficoltà per decisione del pannello Match). Un errore in partita libera non ha MWC: viene solo contato, «non quantificato».</li>
<li>L'<strong>MWC recuperabile</strong> di una famiglia è la somma di ℓ − d sui suoi errori: la frequenza della famiglia moltiplicata per la sua perdita media oltre la difficoltà. È ciò che recupererebbe giocando quelle posizioni come il giocatore di riferimento. Una famiglia di errori evitabili sale, una famiglia di posizioni in cui tutti sbagliano scende.</li>
<li>L'<strong>intervallo al 95 %</strong> accompagna ogni cifra. Una famiglia entra nel piano a partire da <strong>5 errori</strong> e da un intervallo interamente sopra lo zero; il piano è ordinato per il limite inferiore dell'intervallo, così che, a pari recuperabile, passi avanti la famiglia meglio accertata. Le altre sono nominate sotto la tabella, <strong>da confermare</strong>, senza rango: il piano non la spinge verso il rumore.</li>
</ul>
<p>Ogni famiglia propone tre azioni: <strong>Studiare</strong> apre la coda di studio sulle sue posizioni, lo scarto maggiore dal giocatore di riferimento per primo; <strong>Quiz</strong> avvia l'esercizio Decisione del pannello Allenamento su venti di esse; <strong>Anki</strong> ne fa un mazzo di carte. Sopra la tabella, <strong>Quiz sulle prime tre famiglie</strong> estrae venti posizioni da quelle delle tre famiglie di testa. Il piano segue il filtro del pannello: imposti il giocatore per ottenere <em>il suo</em> piano. Da riga di comando: <code>blunderdb stats plan</code> (vedere stats — Errori ricorrenti).</p>
<h5>PR mobile sulle ultime N decisioni</h5>
<p>Una riga di valori PR (o MWC) calcolati sulle ultime <em>N</em> decisioni (N = 5, 10, 50, 100, 250, 500, 1000) permette di misurare la tendenza recente. I valori in grigio corrispondono a un N superiore al numero di decisioni disponibili.</p>
<p>Cliccando su un valore si caricano le ultime <em>N</em> posizioni corrispondenti.</p>
<h5>Top blunder</h5>
<p>L'elenco dei 10 errori peggiori (o MWC cost), ordinati per magnitudo decrescente. Cliccando su una riga si carica la posizione interessata nel pannello di analisi.</p>
<h4>Scheda Progressione</h4>
<p>La scheda <strong>Progressione</strong> presenta l'evoluzione del livello nel tempo.</p>
<p>In testa alla scheda, un <strong>obiettivo</strong>: «PR &lt; 5 entro dodici settimane». Un traguardo, una scadenza, e una tendenza che dice dove si va — nulla di più. Un obiettivo che si mettesse a dare voti, a congratularsi o a ricordare sarebbe un'altra funzione, non questa.</p>
<p>Il pulsante <strong>Proporre</strong> suggerisce un traguardo a partire dal livello attuale: il limite inferiore della fascia in cui siete, cioè l'ingresso in quella successiva. Proporre «un po' meglio» non si ancorerebbe a nulla; proporre un gradino dice qualcosa — passare da intermedio ad avanzato si vede e si racconta.</p>
<p>La <strong>tendenza</strong> è un adattamento ai minimi quadrati sul PR dei vostri incontri, proiettato alla scadenza. Rifiuta di pronunciarsi sotto tre incontri: tracciare una retta fra due punti sarebbe un'affermazione insostenibile. E la frase lo dice ogni volta — <em>una tendenza non è una previsione</em>.</p>
<p>L'obiettivo è memorizzato nei <strong>metadati del database</strong>, non nella configurazione: riguarda quella biblioteca, quindi segue il file anziché la macchina. Nessun cambiamento di schema: <code>metadata</code> è già una tabella di chiavi e valori, leggibile da <code>blunderdb info</code> come dal demone.</p>
<h5>Grafico a linee per torneo</h5>
<p>Un grafico a linee visualizza il PR (o MWC) per ciascun torneo (asse X: ordine dei tornei, asse Y: valore della metrica). Bande colorate evidenziano le soglie di livello.</p>
<p>Cliccando su un punto del grafico si apre un menu contestuale con due opzioni:</p>
<ul>
<li><strong>Apri torneo</strong> — apre il torneo nel pannello Tornei.</li>
<li><strong>Apri posizioni</strong> — carica tutte le posizioni del torneo nel pannello di analisi.</li>
</ul>
<h5>Scatter plot per match</h5>
<p>Un grafico a dispersione rappresenta ciascun match (asse X: data, asse Y: PR o MWC). La dimensione del punto è proporzionale al numero di decisioni nel match.</p>
<p>Cliccando su un punto si apre un menu contestuale:</p>
<ul>
<li><strong>Apri partita</strong> — apre il match nel pannello Match.</li>
<li><strong>Apri posizioni</strong> — carica tutte le posizioni del match nel pannello di analisi.</li>
</ul>
<h4>Scheda Errori</h4>
<p>La scheda <strong>Errori</strong> scompone le fonti di errore.</p>
<h5>Errori ricorrenti</h5>
<p>In cima alla scheda, una tabella raggruppa gli errori del filtro corrente — quindi quelli di un solo giocatore quando un giocatore è filtrato — per <strong>piano di gioco</strong> e per <strong>tema</strong>, il più costoso per primo. Risponde alla domanda «dove perdo di più?»: per esempio, «tenuta · troppi blot».</p>
<ul>
<li>Un <strong>errore</strong> è una decisione conteggiata il cui costo raggiunge la soglia <em>Errore</em> della libreria.</li>
<li>Il <strong>piano di gioco</strong> è quello del giocatore di turno, così come lo presenta la scheda Ripartizioni.</li>
<li>Il <strong>tema</strong> di una mossa di pedine è quello nominato dalle regole della frase di spiegazione del pannello Analisi: gammon sottostimato, troppi blot, punto non fatto, troppo passivo. Quello di una decisione di cubo è il senso dell'errore, come nella direzione degli errori di cubo più sotto: raddoppio mancato, raddoppio prematuro, passo sbagliato, presa sbagliata.</li>
<li>Un errore che nessuna regola nomina con sicurezza non viene indovinato: esce dalla classifica e appare a parte, sotto la tabella, in una riga per piano di gioco (« senza tema identificato: N errori, costo X »), cliccabile anch'essa. Questo resto è spesso il più pesante, perché la spiegazione si pronuncia solo a partire da 60 mp, sopra la soglia <em>Errore</em>: classificato con gli altri, guiderebbe una tabella che non direbbe nulla.</li>
<li>Il <strong>costo</strong> è la quota del PR del filtro che il gruppo rappresenta: la formula del PR applicata agli errori del gruppo, rapportata a tutte le decisioni conteggiate. I costi dei gruppi non superano quindi mai il PR.</li>
</ul>
<p>Facendo clic su un gruppo se ne caricano le posizioni, dalla più costosa alla meno costosa. Il tema viene ricalcolato a ogni visualizzazione e non è mai salvato: come il piano di gioco, è un'etichetta derivata, non modificabile. Da riga di comando: <code>blunderdb stats recurring</code> (vedi stats — Errori ricorrenti).</p>
<p>Ogni riga offre tre gesti per passare dall'errore allo studio: <strong>Quiz su questo gruppo</strong> avvia l'esercizio Decisione del pannello Allenamento sulle posizioni del gruppo, <strong>Mazzo Anki</strong> ne fa un mazzo di carte, <strong>Raccolta</strong> le archivia in una nuova raccolta. Sopra la tabella, <strong>Quiz sui miei tre gruppi peggiori</strong> estrae venti posizioni a caso tra quelle dei tre gruppi più costosi. Dalla riga di comando, <code>stats recurring --quiz</code> estrae queste posizioni e <code>--deck</code> crea il mazzo.</p>
<h5>Ripartizione per azione di cubo</h5>
<p>Un diagramma a barre visualizza il PR (o MWC) per ciascun tipo di decisione di cubo: <em>NoDouble</em>, <em>DoubleTake</em>, <em>DoublePass</em>, <em>TooGood</em>. Ogni barra indica inoltre il numero di decisioni e il tasso di blunder in un suggerimento.</p>
<p>Cliccando su una barra si caricano le posizioni corrispondenti a quell'azione di cubo, <strong>solo quelle con un errore</strong> (drill-down).</p>
<h5>Direzione degli errori di cubo</h5>
<p>La ripartizione qui sopra indica <em>quanto</em> costano le decisioni di cubo; questa tabella indica in <em>quale senso</em> sbagliano.</p>
<p>Una posizione di cubo porta due decisioni prese da due giocatori diversi, presentate qui in due righe:</p>
<ul>
<li><strong>Offrire</strong> — il giocatore che detiene il cubo raddoppia o no. I suoi errori sono i <strong>doppi mancati</strong> (bisognava raddoppiare) e i <strong>doppi prematuri</strong> (non bisognava).</li>
<li><strong>Rispondere</strong> — il giocatore a cui il cubo viene offerto prende o passa. I suoi errori sono i <strong>pass errati</strong> (una presa corretta è stata passata) e le <strong>prese errate</strong> (un pass corretto è stato preso).</li>
</ul>
<p>Le due righe restano separate di proposito: un giocatore può benissimo raddoppiare tardi <em>e</em> prendere largo, e un indicatore unico chiamerebbe ciò «equilibrato» perdendo entrambe le metà dell'informazione.</p>
<p>Ogni casella mostra il numero di decisioni; il tooltip dà l'equity perduta cumulata. Fare clic su una casella carica le posizioni corrispondenti. Una casella a zero non è cliccabile.</p>
<div class="admonition note">
<p>Questa tabella conta decisioni, non emette giudizi. A partire da quale scarto una tendenza meriti di essere nominata dipende dalla numerosità e da un punto di riferimento, che non sono dati del motore.</p>
</div>
<h5>Confronto Checker / Cube</h5>
<p>Un diagramma comparativo affianca il PR delle mosse di pedine e quello delle decisioni di cubo. Cliccando su una barra si caricano le posizioni del sottoinsieme con errore.</p>
<h5>Istogramma delle magnitudo d'errore</h5>
<p>Un istogramma distribuisce gli errori in base alla loro magnitudo in millesimi di punto (fasce: 0–5, 5–10, 10–25, 25–50, 50–100, ≥ 100). Cliccando su una barra si caricano le posizioni della fascia.</p>
<h4>Allenamento nel tempo</h4>
<p>La scheda <strong>Allenamento</strong> affianca, sulle stesse finestre di calendario — la <strong>settimana</strong> o il <strong>mese</strong>, a scelta — tre serie che misurano il progresso in tre modi:</p>
<ul>
<li>il <strong>PR del quiz</strong>: quello delle sessioni dell'esercizio Decisione del pannello Allenamento, ponderato per il numero di decisioni giudicate. È calcolato sulla scala del PR reale, quindi è confrontabile con esso;</li>
<li>il <strong>PR delle partite</strong> del filtro corrente, ponderato per il numero di decisioni;</li>
<li>la <strong>ritenzione Anki</strong>: la quota di ripassi di carte già apprese valutati <em>Difficile</em> o meglio, letta sull'asse destro (in %).</li>
</ul>
<p>Sotto il grafico, il <strong>PR del quiz per piano di gioco</strong> elenca le decisioni del quiz tratte da posizioni della libreria, i piani peggiori per primi, con il PR dell'ultima finestra in cui il piano è stato giocato.</p>
<p>Tra parentesi, il numero di decisioni o di ripassi dietro ogni valore: una finestra senza campioni non ha valore — un trattino, non uno zero. Il filtro restringe solo le partite; il diario del quiz e quello di Anki sono vostri e non portano un giocatore. Non si registra nient'altro: le tre serie sono rilette dai diari esistenti. Da riga di comando: <code>blunderdb stats training</code> (vedi stats — Errori ricorrenti).</p>
<h4>Scheda Ripartizioni</h4>
<p>La scheda <strong>Ripartizioni</strong> divide le stesse decisioni che contano le cifre globali lungo quattro assi. Nessuno di essi ridefinisce cosa conta come decisione: sarebbe un secondo PR con lo stesso nome.</p>
<ul>
<li><strong>Per fase di gioco</strong> — apertura, mediogioco, corsa, uscita delle pedine. È ciò che risponde a «il mio PR in corsa contro il mio PR in contatto». L'etichetta è calcolata dalla tavola (vedi Pannello Ricerca); una base le cui fasi non sono mai state calcolate mette tutto sotto <em>Non classificata</em>, e <code>blunderdb repair</code> la riempie.</li>
<li><strong>Per piano di gioco</strong> — corsa, blitz, ancora, backgame, blocco contro blocco… È la ripartizione per cui il classificatore esiste: «dove perdo di più?», piano per piano. La stessa etichetta derivata della fase, le stesse riserve, e <code>blunderdb repair</code> la riempie allo stesso modo.</li>
<li><strong>Per etichetta</strong> — i <code>#parola</code> scritti nei commenti. Una posizione può portarne più d'una: <strong>queste righe non sommano al totale</strong>, e il pannello lo dice sotto la tabella. Un'etichetta qualifica, non partiziona.</li>
<li><strong>Per punteggio</strong> — i punti mancanti a entrambi i campi, letti dal lato del giocatore di turno, quindi dal lato di chi decide. La riga <em>Money</em> è la partita a soldi. Una cella con meno di dieci decisioni è <strong>in grigio con il suo effettivo visibile</strong> invece che nascosta: troppo poco per essere letta, ma l'omissione resta verificabile.</li>
</ul>
<p>Ogni riga porta il suo <strong>intervallo di confidenza al 95 %</strong> (colonna <em>IC 95 %</em>), che sostituisce la soglia fissa di dieci decisioni: dice quanto vale un PR, non solo quante decisioni lo sostengono. Ricampiona i <strong>match</strong> della selezione, non le decisioni né le partite: le partite di un match condividono avversario, sessione e stanchezza, e contarle come indipendenti darebbe un intervallo troppo stretto; senza filtro giocatore, i due posti di un match formano una sola unità. Una riga che poggia su un solo match non ha intervallo ed è attenuata, anche con molte decisioni. Per lo studio: una famiglia di posizioni il cui intervallo resta sopra il vostro PR globale è una debolezza accertata; una riga con un intervallo ampio non giustifica ancora un piano di lavoro. Il PR globale della dashboard porta lo stesso intervallo, sotto la sua scheda.</p>
<div class="admonition note">
<p>La partita Crawford non è distinta: blunderDB non registra questo indicatore su una posizione. L'effetto pratico è modesto — una partita Crawford non ha alcuna decisione di cubo — ma l'omissione è reale e vale meglio scriverla che lasciarla indovinare.</p>
</div>
<h4>Studio e gioco reale</h4>
<p>Il comando <code>blunderdb list --type study --days 30</code> mette tre numeri uno accanto all'altro, piano per piano: quante <strong>posizioni distinte</strong> sono state ripassate nel periodo, quale era il PR <strong>prima</strong>, quale è il PR <strong>da allora</strong>.</p>
<p>Tre numeri, e nessun quarto. <strong>Non c'è colonna di guadagno né freccia</strong>, perché qui nulla controlla nulla: il giocatore può aver incontrato avversari più forti, cambiato formato, o semplicemente giocato più corse questo mese. L'accostamento è del lettore; una colonna che annunciasse un effetto affermerebbe una causalità che questi dati non portano. I numeri, invece, sono esatti.</p>
<p>I ripassi sono contati in <strong>posizioni distinte</strong>: una carta ripassata quattro volte nel mese è una posizione studiata, e contare le ripetizioni farebbe sembrare un mese di sgobbata un mese di copertura. Le decisioni del PR, invece, sono contate tutte — ciascuna è stata presa una volta. Un PR che poggia su meno di dieci decisioni mostra <code>—</code>, con il suo campione visibile accanto.</p>
<h4>Scheda Giocatori</h4>
<p>Le cinque schede precedenti descrivono <strong>un</strong> giocatore; la scheda <strong>Giocatori</strong> li confronta tutti. Mostra una riga per giocatore del database, il che risponde all'esigenza di un organizzatore che segue un'intera competizione anziché un giocatore in particolare.</p>
<p>Colonne, nell'ordine:</p>
<table>
<thead>
<tr>
<th>Colonna</th>
<th>Significato</th>
</tr>
</thead>
<tbody>
<tr>
<td>Giocatore</td>
<td>Il nome <strong>così come figura nei match</strong>. Un giocatore registrato con due grafie compare su due righe, a meno che una sia un alias dell'altra (sezione Corpus delle impostazioni): la riga porta allora il nome canonico.</td>
</tr>
<tr>
<td>Match</td>
<td>Numero di match disputati nel periodo considerato.</td>
</tr>
<tr>
<td>V–S</td>
<td>Vittorie e sconfitte. Un match incompiuto (registro troncato, abbandono) non conta né l'una né l'altra: V + S può quindi essere inferiore al numero di match.</td>
</tr>
<tr>
<td>Decisioni</td>
<td>Numero di decisioni conteggiate — il denominatore del PR. È la colonna che dice quanto valgono i tassi vicini: un PR calcolato su dodici decisioni non significa nulla.</td>
</tr>
<tr>
<td>PR</td>
<td>Performance Rating globale.</td>
</tr>
<tr>
<td>PR pedine, PR cubo</td>
<td>Il PR ripartito per tipo di decisione.</td>
</tr>
<tr>
<td>Snowie</td>
<td>Snowie Error Rate (vedere Appendice: modello statistico — allineamento XG / gnuBG / blunderDB).</td>
</tr>
<tr>
<td>Blunder</td>
<td>Numero di errori che raggiungono la soglia di blunder della libreria (0,100 EMG per impostazione predefinita).</td>
</tr>
<tr>
<td>Fortuna</td>
<td>Fortuna media per lancio, in millesimi di punto, con segno: positiva se i dadi sono stati favorevoli.</td>
</tr>
</tbody>
</table>
<p>Utilizzo:</p>
<ul>
<li><strong>Ordinare</strong> — fare clic su un'intestazione di colonna. La tabella si apre ordinata per PR crescente, con il miglior giocatore in testa. I giocatori di cui nulla è stato misurato restano in fondo qualunque sia il verso dell'ordinamento: uno zero per mancanza di dati non è una prestazione perfetta.</li>
<li><strong>Aprire il dettaglio di un giocatore</strong> — fare clic su una riga. Il giocatore viene selezionato nella barra dei filtri e la visualizzazione passa alla scheda Cruscotto.</li>
<li><strong>Restringere il periodo</strong> — i filtri di date, tornei e lunghezza dei match si applicano normalmente, il che consente di delimitare la tabella alle date di una competizione.</li>
<li><strong>Confrontare due giocatori</strong> — spuntate la casella della prima colonna su due righe. Sopra la tabella compare un blocco che mette i loro indicatori a confronto; spuntare un terzo giocatore sostituisce il più vecchio dei due. La casella non seleziona la riga: spuntare confronta, cliccare apre il dettaglio.</li>
<li><strong>Vedere le posizioni in cui uno ha giocato meglio</strong> — il pulsante del blocco di confronto elenca le posizioni che entrambi i giocatori hanno dovuto giocare e in cui uno ha giocato bene (sotto la soglia di Errore della biblioteca) e l'altro no, la differenza maggiore per prima; un giocatore è giudicato sulla sua peggiore giocata della posizione. <em>Apri queste posizioni</em> le carica nella vista di analisi. La riga di comando e il server danno lo stesso elenco (<code>stats contrast</code>).</li>
<li><strong>Valutare prima le mosse</strong> — l'elenco confronta solo le mosse il cui errore è valutato, e nessuna importazione le valuta: la valutazione si fa su richiesta. Finché ne restano da valutare, il blocco ne indica il numero con un pulsante <em>Valuta le mosse</em>; la riga di comando fa lo stesso con <code>repair --move-errors</code>.</li>
</ul>
<p>In quel blocco <strong>solo i tassi ricevono un verdetto</strong>, e il migliore dei due è in grassetto. Tre indicatori non ne ricevono mai, e vale la pena dire perché. La <strong>fortuna</strong> non è una qualità: un giocatore più fortunato non è migliore. <strong>Partite, bilancio e decisioni</strong> dicono quanto valgono i tassi, ma metterli in competizione farebbe vincere chi ha semplicemente giocato di più. Il <strong>numero di blunder</strong> non si confronta grezzo — dodici su mille decisioni valgono più di dieci su cento —, perciò il blocco aggiunge una riga <em>Blunder / 100 dec.</em> che invece si confronta, e lascia il conteggio accanto come contesto.</p>
<p>Un pari non è una vittoria: non è messo in grassetto da nessuna parte. Un tasso senza nulla dietro si mostra come «—» e non decide nulla.</p>
<div class="admonition note">
<p>In questa scheda l'elenco <strong>Giocatore</strong> e la scelta del <strong>tipo di decisione</strong> sono disattivati: la tabella mostra tutti i giocatori e ripartisce già le decisioni di pedine e di cubo in colonne distinte.</p>
</div>
<p>Tre viste di corpus figurano nella scheda <strong>Corpus</strong> del pannello, da riga di comando e tramite l'API del demone (vedi stats — Errori ricorrenti): il <strong>testa a testa</strong> di due giocatori (match in comune, PR di ciascuno, bilancio), il <strong>PR per finestra di calendario</strong> scorrevole (1, 3, 6 o 12 mesi) e la <strong>classifica</strong> per PR dei giocatori che hanno almeno un dato numero di decisioni contate. La scheda Corpus calcola ogni vista su richiesta, sotto il filtro corrente. Un <strong>filtro di provenienza</strong> (campi <em>Motore</em> e <em>Profondità min.</em> della barra dei filtri) limita le statistiche alle decisioni analizzate così; riguarda ogni decisione, e le cifre che tocca sono allora ricalcolate dalle decisioni. Il testa a testa e il PR per finestra non lo accettano: finché è attivo, rifiutano di calcolarsi. Come il resto della barra, viene conservato da una sessione all'altra.</p>
<div class="admonition important">
<p>Un trattino («—») segnala un valore <strong>mai misurato</strong>, da non confondere con zero. È in particolare il caso della colonna Fortuna per ogni match importato prima della versione 2.15.0 dello schema: la fortuna non veniva allora conservata, e nulla ne consente la ricostruzione a posteriori. Reimportare il file di origine non basta: l'importazione riconosce un duplicato e applica solo i contrassegni di studio appena attivati (il rapporto ne dà il numero). Occorre eliminare il match e poi reimportarlo. I formati che non la trasportano (BGF, Jellyfish <code>.mat</code>) non la forniranno mai.</p>
</div>
<h4>Regola di aggregazione</h4>
<div class="admonition important">
<p>Il PR di un torneo (o di un qualsiasi sottoinsieme) è calcolato con la regola <strong>somma/somma</strong> — mai come media dei PR individuali dei match.</p>
<p>Formula:</p>
<pre class="math">PR_&#123;torneo&#125; = 500 \\times \\frac&#123;\\sum_&#123;i&#125; \\text&#123;errore&#125;_i&#125;&#123;\\text&#123;numero totale di decisioni&#125;&#125;</pre>
<p><strong>Esempio:</strong> un giocatore disputa due match in un torneo —</p>
<ul>
<li>Match A: 10 decisioni, 0,100 di equità persa → PR = 5,0</li>
<li>Match B: 90 decisioni, 0,540 di equità persa → PR = 3,0</li>
</ul>
<p>Media ingenua dei PR: (5,0 + 3,0) / 2 = <strong>4,0</strong> <em>(errato)</em></p>
<p>Regola somma/somma: 500 × 0,640 / (10 + 90) = <strong>3,2</strong> <em>(corretto)</em></p>
<p>La regola somma/somma è l'unica che gestisce correttamente la variazione di lunghezza dei match (un match a 21 punti pesa più di un match a 1 punto).</p>
</div>
<h4>MWC: limitazioni</h4>
<ul>
<li>Per impostazione predefinita, il MWC cost è calcolato a partire dalla <strong>MET Kazaross-XG2</strong>, tabella di riferimento de facto nel backgammon competitivo. I risultati non sono direttamente confrontabili con software che usano altre MET. È la stessa tabella, letta dallo stesso punto d'ingresso, di quella di cui il valutatore integrato si serve per le sue decisioni di cubo al punteggio: le statistiche e il motore non possono divergere su questo punto. Fornisce i propri valori fino a 25 punti da fare per parte; oltre, è prolungata da una tabella di Zadeh calcolata come quella di GNUbg, fino a 64.</li>
<li>Le posizioni <em>money-game</em> (senza punteggio di match) sono <strong>escluse</strong> dal calcolo MWC. Se il database contiene molte posizioni money-game, il MWC cost potrebbe essere sottostimato o non disponibile.</li>
<li>Il MWC cost è cumulativo sull'intero set di dati filtrato — non è un indicatore per singola decisione. Misura l'impatto totale dei tuoi errori sulle tue probabilità di vittoria.</li>
</ul>
<h3>Pannello Eval</h3>
<p>Il pannello <strong>Eval</strong> (<em>CTRL-E</em>) valuta in tempo reale qualunque posizione si trovi sul tavoliere; su una posizione di bearoff si specializza e calcola inoltre l'EPC (Effective Pip Count). Si attiva premendo <em>CTRL-E</em>, facendo clic sulla scheda Eval del pannello inferiore, oppure eseguendo il comando <code>eval</code>; anche <code>epc</code>, il suo vecchio nome, lo apre. Il pannello si è chiamato <em>EPC</em>, poi <em>Bearoff</em>, prima di diventare <em>Eval</em> — è dunque qui che va cercato ciò che una versione precedente chiamava il pannello Bearoff, il nome non designando più che la scheda di configurazione delle tabelle di uscita.</p>
<p>Il pannello mostra sempre l'<strong>unica decisione</strong> che la posizione posta sul tavoliere richiede — mai due alla volta — e i fatti che l'accompagnano. Ogni grandezza si legge nell'asse che le conviene anziché in un asse unico imposto: la probabilità di vittoria, di gammon, di backgammon e l'equità cubeless di ciascun giocatore, calcolate <em>prima del lancio</em>, si leggono <strong>per giocatore</strong> (basso, alto, poi Δ), a sinistra della decisione di cubo, quando nessun dado è impostato. I fatti e la decisione restano fianco a fianco: la decisione di cubo non finisce mai sotto i numeri che la giustificano, qualunque siano la lingua dell'interfaccia e la posizione sul tavoliere. Non appena dei dadi sono impostati, questi stessi valori <em>prima del lancio</em> cambiano asse: si leggono <strong>al tiro</strong>, in testa alla lista delle mosse candidate, sotto forma di una riga in corsivo <em>prima del lancio</em> — non una mossa candidata in più, ma un riferimento rispetto al quale leggere ogni mossa. Lo scarto tra questa riga e una mossa contiene la fortuna del tiro, mai il merito della mossa, e la riga non porta quindi alcuna colonna di errore. Su una posizione di bearoff puro, una seconda tabella, sempre <strong>per giocatore</strong> e sempre presente, dadi impostati o meno, porta l'EPC, il pip count, il wastage, il numero medio di lanci e la deviazione standard; queste cinque colonne non migrano mai. Le due tabelle sono impilate e condividono la stessa griglia di colonne: stessi bordi, stessi riferimenti di colonna, una sola colonna di pallini — si leggono come un unico oggetto a due piani. Il pulsante <em>Aggiungi al database</em>, il badge di regime, l'attribuzione del motore (vi figura anche la profondità dell'ultima valutazione) e la casella <em>Sfida</em> formano una fascia a parte, allineata a destra sopra le tabelle.</p>
<p>Solo la lista delle mosse candidate scorre — anche la riga <em>prima del lancio</em> resta fissata sopra di essa; il resto del pannello (fatti, badge, decisione di cubo) resta sempre visibile, senza alcuna regolazione particolare della dimensione del pannello.</p>
<p>La tabella dei fatti e la decisione sono calcolate da gammonNet, integrato, senza XG né gnubg. Il calcolo segue la posizione senza mai bloccare l'interfaccia: una profondità 0-ply viene mostrata immediatamente a ogni gesto, poi, dopo mezzo secondo di immobilità, una valutazione più profonda (2 ply per impostazione predefinita, regolabile nella scheda <em>gammonNet</em> della configurazione) la sostituisce in background — qualsiasi nuovo gesto annulla questo calcolo di fondo. La profondità mostrata nella fascia dei badge, o all'interno del badge di regime su una posizione di corsa, è sempre quella che ha effettivamente prodotto il numero mostrato, mai quella richiesta; non si ripete su ogni riga, poiché una valutazione in diretta condivide la stessa profondità per tutte le mosse. L'equità delle mosse candidate e della decisione di cubo segue il punteggio della posizione: in money game è espressa in punti, a un punteggio di match in <strong>equità normalizzata</strong> — la stessa scala di XG e GNU Backgammon, in cui vincere il valore del cubo corrente vale +1 e perderlo −1 — mai mescolate in una stessa tabella. L'intestazione della colonna lo dichiara esplicitamente invece di lasciare indovinare la scala: «Equity (money)» in money game, «Equity (match)» a un punteggio di match. Tiene conto del <strong>cubo vivo</strong>: la ricerca valorizza ogni posizione finale con il modello di cubo (Janowski, efficienza misurata) nello stato del cubo della posizione, come fanno XG e GNU Backgammon nella valutazione <em>cubeful</em>. È ciò che rende visibili al punteggio gli effetti gammon-go e gammon-save — a 4-away/2-away, il giocatore in svantaggio gioca 8/2 6/2 su un 6-4 di apertura perché il suo raddoppio anticipato darà al gammon il valore del match, cosa che una valutazione senza cubo non può vedere. La riga <em>prima del lancio</em>, invece, resta un'equità <strong>cubeless</strong>: è un fatto della posizione, non una decisione. La valutazione non viene mai registrata: è un calcolo, non un'analisi. Cliccare una mossa candidata la mostra sul tavoliere sotto forma di frecce, esattamente come nel pannello Analisi. Il discreto pulsante <strong>?</strong>, nella fascia dei badge, conduce al repository del motore <code>gammonNet &lt;https://github.com/kevung/gammonNet&gt;</code>_; l'attribuzione completa (rete Strehl, configurazione gammonNet) figura nei Ringraziamenti dell'aiuto.</p>
<p>Il pannello <strong>rolla</strong> la posizione come il pannello Analisi (vedi Rollout): <em>Ctrl+clic</em> e <em>Maiusc+clic</em> selezionano le mosse, il clic destro apre lo stesso menu — rollout, copia della posizione e della valutazione, o della posizione e delle mosse selezionate —, <em>r</em> avvia o ferma il rollout e <em>Esc</em> lo annulla; una barra sottile segue le partite giocate, e ogni mossa rollata porta il suo risultato nella colonna <strong>Rollout</strong> (senza dadi, sotto la decisione di cubo). Poiché la tavola è una bozza, il rollout non viene mai registrato: si gioca in memoria, senza database aperto come su un database in sola lettura, e il suo risultato è mostrato solo finché la tavola resta la stessa. Un rollout avviato prima di una modifica della tavola non viene mostrato sulla nuova posizione.</p>
<p>Il pulsante <strong>Aggiungi al database</strong>, in testa alla fascia dei badge, registra nel database la posizione presente sul tavoliere; <em>CTRL-S</em>, il comando <code>w</code> e il pulsante <em>Salva posizione</em> della barra degli strumenti fanno lo stesso. Viene scritta solo la posizione, mai la valutazione mostrata; se l'analisi automatica gammonNet è attiva, il suo lotto parte subito dopo, come dopo un'importazione. La barra di stato annuncia il numero della posizione, anche quando era già nel database: viene allora contrassegnata come importata singolarmente, e il filtro <em>Importata singolarmente</em> (<code>s i</code>) la ritrova. Il pannello resta aperto sullo stesso tavoliere, che resta una tavola di prova: si può spostare una pedina e aggiungere la variante, e uscire dal pannello riporta a ciò che si stava studiando. Il pulsante è disattivato finché non è aperto alcun database, o finché la posizione non può essere registrata (per esempio la posizione iniziale del pannello, in cui il giocatore in alto ha portato fuori tutte le sue pedine); il suo suggerimento ne indica il motivo.</p>
<p>L'utente modifica la posizione delle pedine sull'intero tavoliere, esattamente come in modalità di modifica: il clic sinistro posiziona una pedina del giocatore in basso, il clic destro una pedina del giocatore in alto. La seconda tabella, quella della corsa, compare solo quando la posizione ottenuta è un bearoff puro (tutte le pedine di entrambi i giocatori nella propria tavola); su qualsiasi altra posizione risponde soltanto la tabella delle quattro colonne comuni (vittoria, gammon, backgammon, cubeless), e la decisione riguarda le pedine o un cubo generico a seconda che dei dadi siano impostati.</p>
<p>In ciascuna tabella dei fatti, una riga per giocatore — contrassegnata dal suo pallino colorato, con il giocatore nero sempre in basso. La prima porta, finché nessun dado è impostato, la vittoria, il gammon, il backgammon (probabilità, senza il segno %) e l'equità cubeless del giocatore; la seconda, su una posizione di bearoff e dadi impostati o meno, l'EPC, il pip count, il wastage (differenza tra l'EPC e il pip count), il numero medio di lanci e la deviazione standard. Quando entrambi i giocatori hanno valori da confrontare, una riga <strong>Δ</strong> fornisce le differenze <em>con segno</em> (basso − alto: negativo quando il giocatore nero è in vantaggio). Fuori da una posizione di corsa, impostare dei dadi fa quindi scomparire le tabelle dei fatti stesse: le quattro colonne che portavano hanno appena cambiato asse, al tiro, in testa alla lista delle mosse.</p>
<p>La decisione di cubo ha sempre la stessa forma, qualunque sia l'origine dei numeri — tabella esatta, regime valutato o valutazione gammonNet ordinaria: <strong>una riga per opzione</strong>, nell'ordine <em>nessun raddoppio</em>, <em>raddoppio/prende</em>, <em>raddoppio/passa</em>, con la sua equità nel riferimento della posizione e il suo scarto rispetto all'opzione migliore. L'ordine non cambia mai, a differenza della lista delle mosse: le tre opzioni portano un nome, ed è quindi il nome che si legge, non il rango. La migliore si riconosce dalla sua evidenziazione e dalla sua cella di scarto lasciata vuota. Quando il cubo è già stato girato, le opzioni si leggono <em>nessun riraddoppio</em>, <em>riraddoppio/prende</em>, <em>riraddoppio/passa</em>.</p>
<p>Un'ultima riga dà il <strong>verdetto</strong>. Assume quattro valori: <em>nessun raddoppio</em>, <em>raddoppio, prende</em>, <em>raddoppio, passa</em> e <em>troppo forte per raddoppiare</em>, quest'ultimo quando giocare la posizione rende più che incassare il punto: raddoppiare sarebbe allora un errore per la ragione opposta a quella del semplice <em>nessun raddoppio</em>. È anche l'unico punto in cui il pannello dice che <strong>non</strong> c'è verdetto, anziché lasciar credere a un calcolo in corso:</p>
<ul>
<li><em>nessuna decisione</em> — il regime non ne ha diritto; il verdetto di cubo non viene mai stimato (vedere il badge <em>stimato</em>);</li>
<li><em>non valutabile a questo punteggio</em> — il motore rifiuta la posizione, tipicamente un punteggio fuori dall'orizzonte della tabella di equità di match, cioè un lato a più di 64 punti da fare;</li>
<li><em>cubo dell'avversario</em> e <em>cubo morto (Crawford)</em> — il cubo non può essere girato. Le equità restano visualizzate, a titolo indicativo, ma nessuna opzione porta uno scarto: un errore è ciò che costa una scelta, e qui non c'è scelta.</li>
</ul>
<p>In money game, le regole <strong>Jacoby</strong> e <strong>Beaver</strong> attive sulla posizione compaiono sotto la tabella del cubo, in piccoli badge accanto al verdetto che modificano: il verdetto «no double» di una posizione sotto la regola Jacoby non è lo stesso calcolo di quello senza di essa, e nient'altro sullo schermo lo indicava.</p>
<p>Con la regola <strong>Beaver</strong>, la riga <em>raddoppio, accetta</em> vale la migliore risposta dell'avversario diversa dall'abbandono: accettare, oppure fare <em>beaver</em> — ribattere subito il raddoppio tenendo il cubo, giocando così la partita al quadruplo della posta —, e chi ha raddoppiato risponde allora con un <em>raccoon</em> (otto volte la posta, cubo dalla sua parte) quando gli conviene. Il verdetto e gli errori si leggono su questa riga, come senza la regola; a un punteggio di match la regola non si applica.</p>
<p>Un terzo distintivo, <strong>Cubo max</strong>, compare quando l'identificatore di origine limita il cubo — sia a un punteggio di incontro sia nel money game. Quello non descrive il calcolo mostrato sopra: il valutatore integrato non modella un tetto, quindi il verdetto è quello di un cubo libero. È proprio per questo che il distintivo c'è: un cubo limitato è l'unica ragione visibile per cui blunderDB ed eXtreme Gammon possono annunciare due verdetti diversi sulla stessa posizione.</p>
<p>Il <strong>giocatore al tiro</strong> e la <strong>posizione del cubo</strong> si modificano direttamente sul tavoliere, come in modalità di modifica: cliccare il rettangolo bearoff/punteggio di un giocatore gli assegna il tiro; cliccare il cubo lo fa ruotare centrato → posseduto in basso → posseduto in alto (clic destro in senso inverso). Il valore del cubo resta fissato — in money game le equità sono espresse in unità del cubo corrente, conta solo il suo proprietario. L'analisi viene ricalcolata immediatamente. In regime stimato, il badge stesso è cliccabile e apre direttamente la scheda <em>Bearoff</em> della configurazione; il suo tooltip spiega perché (verdetto di cubo non stimabile, <code>ADR-0009 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0009-race-win-chances-are-read-or-convolved-cube-verdicts-are-never-estimated.md&gt;</code>__) e come estendere il dominio esatto.</p>
<p>Anche il <strong>punteggio</strong> si modifica direttamente sul tavoliere, come in modalità di modifica: il clic sinistro sul rettangolo del punteggio di un giocatore decrementa il suo numero di punti da fare, il clic destro lo incrementa. Uscire dal punteggio <em>money</em> (-1, -1) modificando un solo campo allinea automaticamente l'altro campo sullo stesso valore anziché lasciare un punteggio incoerente. Su una posizione di bearoff in regime <em>esatto</em>, passare da un punteggio money a un punteggio di match lascia la probabilità di vittoria così com'è (una lettura dal database, valida qualunque sia il riferimento) ma fa passare l'equità e il verdetto di cubo mostrati a quelli del regime <em>valutato</em> — essendo la tabella esatta money per costruzione, non sa rispondere alla domanda posta al punteggio. Il badge diventa allora composito (« esatto (vittoria) · valutato (cubo) ») per dirlo esplicitamente.</p>
<p>I <strong>dadi</strong>, infine, si modificano allo stesso modo, e sono loro a decidere la domanda posta: dadi impostati fanno una decisione di pedine (l'elenco delle mosse candidate), nessun dado una decisione di cubo. Un clic sinistro su un dado ne aumenta il valore (il 6 torna a 1), un clic destro lo diminuisce (l'1 torna a 6); cliccare un dado su un tavoliere che non ne ha ne imposta due in un colpo solo — un dado singolo non sarebbe né una decisione di pedine né una decisione di cubo. Cliccare il rettangolo di un giocatore toglie i dadi per porre una domanda di cubo, e il clic successivo su un dado li rimette com'erano.</p>
<p><em>BACKSPACE</em>, o <em>Cancella la posizione</em> nel menu di un clic destro fuori dal tavoliere, cancella la posizione: tavoliere vuoto, punteggio money (-1, -1), nessun dado impostato — valori propri del pannello Eval, diversi da quelli usati in modalità di modifica (7 ovunque, dadi 3-1), per restare coerenti con ciò che il pannello mostra per impostazione predefinita. <em>Posizione iniziale</em>, nello stesso menu, dispone le pedine di una partita nuova su questi stessi valori.</p>
<h4>Matrice del cubo</h4>
<p>Una decisione di cubo non è una proprietà della dama. Le stesse pedine, lo stesso conteggio dei pip, si raddoppiano a 2-away/4-away e non si raddoppiano a 4-away/2-away; chi ha imparato la risposta money ha imparato una sola casella di una griglia. Il pannello Eval mostra la casella che la posizione porta; la <strong>matrice del cubo</strong> mostra l'intera griglia.</p>
<p>Il comando <code>cm</code> la apre sulla posizione visualizzata. Ogni casella dà il verdetto a un punteggio: la riga è il numero di punti che restano da fare al giocatore di turno, la colonna quelli dell'avversario. I quattro verdetti si scrivono <em>ND</em> (niente raddoppio), <em>DP</em> (raddoppio, presa), <em>DR</em> (raddoppio, rifiuto) e <em>TB</em> (troppo buono); una casella rifiutata dal motore porta un punto interrogativo e spiega perché al passaggio del mouse, che dà anche le tre equità della casella. Sono proposte tre lunghezze di incontro: 5, 7 e 9 punti.</p>
<p>La casella del punteggio che la posizione porta davvero è incorniciata, e le sue intestazioni di riga e di colonna sottolineate: la lettura parte da lì, «la mia casella, e ciò che la circonda». Lo è non appena i due punteggi <em>away</em> della posizione entrano nella griglia mostrata; cambiare lunghezza la sposta o la toglie. Una posizione money, la partita Crawford o un <em>away</em> oltre la griglia non ne indicano alcuna: non c'è casella da mostrare, e mostrarne una approssimativa sarebbe falso.</p>
<p>Il punteggio della posizione è sostituito da quello di ogni casella; il suo <strong>cubo</strong> è conservato. La griglia risponde a quale punteggio girerei <em>questo</em> cubo, non a ciò che farebbe una posizione centrata. È post-Crawford da un capo all'altro: durante la partita Crawford il cubo non è in gioco, e una colonna di «non potete raddoppiare» non direbbe nulla sulla posizione.</p>
<p>Ogni casella è una ricerca a sé. Il motore tiene conto del punteggio — non gioca la stessa partita a 2-away e a 7-away — quindi una sola ricerca riletta attraverso equità di incontro diverse sarebbe falsa esattamente dove il punteggio conta. La griglia arriva prima in 0-ply, poi si ricalcola alla profondità di visualizzazione configurata una volta che la finestra è a riposo: la stessa escalation del resto del pannello, per una griglia da 9 punti che costa circa un secondo e mezzo.</p>
<p>La stessa griglia si calcola fuori dall'interfaccia, con il comando cubematrix della riga di comando.</p>
<h4>Portare una posizione nel pannello Eval</h4>
<p>Il pannello si apre per impostazione predefinita su una posizione di bearoff, ma lo studio parte più spesso da una posizione già in mano. Tre gesti ve la portano:</p>
<ul>
<li><strong>Clic destro sul tavoliere</strong>, in un pannello di analisi o durante la navigazione di un match, poi <em>Valuta questa posizione</em>: il pannello Eval si apre direttamente su questa posizione, così come è visualizzata; <em>Valuta lo specchio di questa posizione</em> ve la apre vista dall'altro lato. Il menu contestuale non compare nel pannello Eval né nel pannello Ricerca, dove il tasto destro serve già a collocare le pedine dell'altro colore.</li>
<li><strong>CTRL-C poi CTRL-V</strong>: copiare la posizione dal pannello di analisi, poi incollarla una volta nel pannello Eval. L'incollaggio accetta anche un identificatore proveniente da altrove — un XGID (eXtreme Gammon, GNU Backgammon, un'altra istanza di blunderDB) o un OGID (OpenGammon): basta che sia negli appunti.</li>
<li><strong>Il comando</strong> <code>import XGID=…</code> (o <code>import OGID=…</code>) per il caso in cui l'identificatore non è negli appunti ma in un messaggio, su un forum letto in un terminale, o prodotto da uno script. È lo stesso verbo di <code>import</code> da solo: senza argomento apre un selettore di file, con un argomento legge l'identificatore. Il percorso è poi identico a quello dell'incollaggio — stessa lettura, stessa deduplicazione, stessa apertura della posizione importata.</li>
</ul>
<p>Un OGID porta solo una posizione: né valutazione, né commento. La posizione arriva quindi senza analisi, esattamente come un XGID nudo, e il valutatore integrato può colmare il vuoto in seguito.</p>
<p>In un OGID, la partita Crawford si riconosce dalla <code>C</code> che segue la lunghezza dell'incontro (<code>7C</code>): senza di essa, un giocatore a un punto dalla meta è dopo la Crawford.</p>
<p>Il tavoliere del pannello Eval è una bozza: la posizione vi arriva senza il suo identificativo di database, in modo che nessuna modifica fatta qui possa riscrivere il record da cui proviene. Tutte le consuete modifiche del tavoliere vi restano disponibili (pedine, cubo, dadi, punteggio), e la valutazione segue ogni modifica.</p>
<p>Nell'altro senso, <em>CTRL-C</em> copia il tavoliere del pannello Eval negli appunti, con un XGID ricalcolato dalle pedine posizionate — quindi incollabile direttamente in eXtreme Gammon o in un'altra istanza di blunderDB. Viaggia soltanto la posizione: la valutazione mostrata dal pannello non è un record del database e non accompagna la copia.</p>
<p>Uscendo dal pannello Eval, la posizione consultata in precedenza viene ripristinata: la bozza non viene mai salvata da sola.</p>
<p>Quando la posizione è un bearoff puro (tutte le pedine di entrambi i giocatori nella propria tavola) e nessun dado è impostato, la decisione di cubo mostra, per il giocatore al tiro:</p>
<ul>
<li>in regime <em>esatto</em>: le equità money (cubeless, senza raddoppio, raddoppio/prende, raddoppio/passa) e il <strong>verdetto di cubo money</strong> (nessun raddoppio, raddoppio/prende, raddoppio/passa o troppo forte per raddoppiare) — fuori dal punteggio di match, vedere sopra per il caso del punteggio,</li>
<li>in regime <em>valutato</em>: le stesse equità e lo stesso verdetto a quattro valori, ma <strong>giocati da gammonNet</strong> (ricerca + modello di cubo Janowski) anziché letti in una tabella — disponibili <strong>anche al punteggio di match</strong>, ciò che il regime stimato non ha mai potuto offrire;</li>
<li>in regime <em>stimato</em>: il verdetto di cubo non viene allora volutamente mostrato — resta disponibile soltanto la probabilità di vittoria, nella tabella dei fatti, accompagnata dal suo margine d'errore.</li>
</ul>
<p>Non appena dei dadi sono impostati su una posizione di corsa, questa decisione di cubo <em>prima del lancio</em> scompare — il tavoliere richiede allora una decisione di pedine, non di cubo — ma la probabilità di vittoria, dal canto suo, resta un fatto della posizione, non una decisione: raggiunge la riga <em>prima del lancio</em> in testa alla lista delle mosse, accanto all'EPC che, invece, resta visualizzato subito a sinistra.</p>
<p>Un badge indica il regime: <strong>esatto</strong> (valore letto in un database two-sided), <strong>valutato · &lt;profondità&gt;</strong> (giocato da gammonNet — la profondità mostrata è quella che ha effettivamente prodotto il numero mostrato), <strong>stimato ± margine</strong>, oppure, al punteggio di match nel dominio esatto, <strong>esatto (vittoria) · valutato (cubo)</strong> — vedere sopra. Il regime esatto prevale ovunque sia disponibile; altrimenti il regime valutato compare non appena ha finito di calcolare, sostituendo sul posto il regime stimato mostrato durante l'attesa. Vedere Metodologia e ipotesi del pannello Eval per la definizione precisa dei tre regimi e delle loro ipotesi.</p>
<p><strong>Allargare il dominio esatto.</strong> La tabella calcolata al primo avvio copre 6 pedine per parte. Due modi per andare oltre, nella scheda <em>Bearoff</em> della configurazione:</p>
<ul>
<li>calcolare una tabella a due lati più ampia — fino a TS-06-15 se la macchina ha la memoria per farlo. La scheda dichiara la dimensione, la memoria e il tempo su questa macchina prima di cominciare, e il calcolo si mette in pausa e si riprende. Un calcolo annullato lascia un file <code>.part</code> che non viene mai letto come una tabella;</li>
<li>indicare un qualsiasi file <code>.bd</code> two-sided di gnubg. Il database con il dominio più ampio prevale automaticamente.</li>
</ul>
<p><strong>Il tavoliere del pannello è una bozza, e viene ricordato.</strong> Uscire dal pannello Eval e tornarci ritrova la posizione su cui lo si è lasciato, non il tavoliere di bearoff predefinito: quello viene servito solo alla prima apertura del pannello in una sessione. Inviare al pannello una posizione dal database prevale su questo ricordo, e <em>BACKSPACE</em> restituisce il tavoliere predefinito in qualsiasi momento. Nulla viene scritto nel database per strada: la bozza non ha identità di posizione, e la sua valutazione viene ricalcolata all'arrivo anziché trasportata.</p>
<p><strong>Modalità sfida.</strong> La casella <em>Sfida</em>, nella fascia dei badge, attiva una modalità di allenamento: a ogni modifica della posizione, i valori di tre zone vengono nascosti (sostituiti da « ··· »); un clic su una zona rivela soltanto quella zona. Senza dadi, si tratta della riga del giocatore in basso, della riga del giocatore in alto e della decisione di cubo — la riga Δ compare solo una volta rivelate entrambe le righe dei giocatori. Il blocco di decisione conserva allora le sue tre righe: sono i suoi valori, il suo verdetto e l'evidenziazione dell'opzione migliore a scomparire, altrimenti l'esercizio si risolverebbe cercando la riga in grassetto. Con dadi impostati su una posizione di corsa, la riga EPC di ciascun giocatore si nasconde come prima, ma la terza zona copre allora la riga <em>prima del lancio</em> e la lista delle mosse <strong>insieme</strong>: essendo la lista ordinata dalla mossa migliore alla peggiore, rivelarla parzialmente darebbe già la risposta. Con dadi impostati fuori da una posizione di corsa, questa stessa zona unica copre da sola tutto ciò che il pannello mostra. Ci si può così allenare a stimare l'EPC di ciascun campo, poi a pronunciarsi sul cubo o sulla mossa da giocare, prima di verificare. L'impostazione viene memorizzata.</p>
<p><strong>Senza database aperto.</strong> Il pannello Eval non ha bisogno di un database: una volta chiusa la schermata iniziale, vi si può impostare, incollare (<em>CTRL-V</em>) e modificare una posizione, valutarla, farne il rollout e copiarla come testo (<em>CTRL-C</em>) o come immagine (<em>CTRL-X</em>, e <em>CTRL-X CTRL-X</em> con la valutazione). Il clic destro nel pannello propone anche <em>Salvare l'immagine (SVG)…</em> e <em>Salvare l'immagine (PNG)…</em>. Anche la scacchiera della scheda Ricerca è una bozza: si modifica e riceve un incolla senza database, e si passa dall'una all'altra senza perdere nulla; avviare la ricerca, invece, richiede un database. Solo <em>Aggiungi al database</em> e <em>CTRL-S</em> richiedono un database nel pannello, e la barra di stato lo segnala. Nessun database viene creato in background. Nel pannello Eval, come nella scheda Ricerca, il comando <code>import XGID=…</code> (o <code>import OGID=…</code>) colloca la posizione sulla scacchiera bozza, senza database. Anche la scheda <strong>Allenamento</strong> si apre senza database: <em>Punteggi</em>, <em>Pedine</em>, <em>Bearoff</em> e <em>Valutazione</em> sono disponibili, <em>Decisione</em> è disattivata con il motivo mostrato, e la sessione terminata non viene registrata (il diario lo indica). La Direzione del torneo richiede un database aperto.</p>
<p>Per chiudere il pannello Eval, premere <em>CTRL-E</em> o passare a un'altra scheda.</p>
<h4>Metodologia e ipotesi del pannello Eval</h4>
<p>Ogni valore mostrato dal pannello si basa su ipotesi precise, enunciate qui in modo esaustivo.</p>
<p><strong>Dominio.</strong> La <em>zona corsa</em> — probabilità di vittoria e verdetto del cubo — tratta solo il bearoff puro: tutte le pedine rimaste dei due giocatori nella loro casa. La posizione è valutata <em>prima del lancio</em>; i dadi eventualmente posati sono ignorati.</p>
<p>I <strong>blocchi EPC</strong>, invece, vanno oltre: un lato ottiene il suo EPC non appena la sua pedina più lontana entra nella tabella a un lato caricata. Con la tabella predefinita (sei punti) è la vecchia regola della casa; con una tabella a otto punti, calcolata dalla scheda <em>Bearoff</em>, un lato con una pedina sull'8 è trattato come gli altri. Nulla è estrapolato: una pedina un punto troppo lontana semplicemente non ha EPC, esattamente come una pedina sul 7 non ne aveva prima. Quando la tabella che ha risposto non è quella a sei punti, il suo nome compare nell'angolo del blocco corsa (« OS-08 ») — senza di esso si leggerebbe « sei » per difetto e si crederebbe il lato interamente rientrato.</p>
<p><strong>Blocchi EPC (sempre esatti).</strong> L'EPC, il numero medio di lanci e la deviazione standard provengono dalla distribuzione esatta del numero di lanci per far uscire tutte le pedine, letta nella base a un lato di GNUbg (da 6 a 10 punti, 15 pedine, calcolata sulla macchina). EPC = lanci medi × 49/6 (49/6 ≈ 8,167 è la media esatta di pip per lancio, doppi contati quattro volte); wastage = EPC − pip count. L'unica idealizzazione è il <em>gioco ottimale a un lato</em>: ogni giocatore minimizza i propri lanci ignorando l'avversario — è la definizione standard dell'EPC.</p>
<p><strong>Probabilità di vittoria, regime esatto.</strong> Lettura diretta nel database two-sided disponibile più ampio (TS-06-06 calcolata al primo avvio, file esterno, o TS-06-11 calcolata dalla scheda <em>Bearoff</em>). Questi database risultano da un'analisi retrograda completa sotto gioco two-sided ottimale di entrambi i campi: nessuna ipotesi supplementare, errore limitato alla quantizzazione (&lt; 0,002 %).</p>
<p><strong>Probabilità di vittoria, regime stimato.</strong> Fuori dal dominio del database: la probabilità si ottiene convolvendo le due distribuzioni one-sided (il giocatore al tiro vince se il suo numero di lanci è inferiore o uguale a quello dell'avversario), poi applicando una correzione polinomiale fissa, calibrata offline rispetto al database TS-06-11. Tre ipotesi:</p>
<ul>
<li><strong>indipendenza</strong> dei due processi di uscita — strutturale in corsa, senza contatto non c'è alcuna interazione;</li>
<li><strong>gioco one-sided ottimale di entrambi i campi</strong> — è <em>l'approssimazione</em>: in realtà il giocatore in svantaggio devia per giocare la varianza e chi conduce per la sicurezza. L'effetto misurato è un bias antisimmetrico (la convoluzione esagera il vantaggio di chi conduce) che la correzione assorbe statisticamente;</li>
<li>la <strong>correzione</strong> è stata calibrata e validata sul dominio dell'oracolo (fino a 11 pedine per giocatore). Errore residuo misurato: deviazione standard 0,05 %, 99º percentile 0,17 %, massimo osservato 0,9 % (in punti di probabilità di vittoria). <strong>Oltre 11 pedine per giocatore, questo limite è estrapolato</strong> — la tendenza è monotona ma nessun oracolo la certifica.</li>
</ul>
<p><strong>Equità e verdetto di cubo (solo regime esatto).</strong> Le equità mostrate sono quelle del <strong>money game, senza Jacoby</strong>, nel riferimento della letteratura del bearoff. Nel dominio ≤ 11 pedine per giocatore i gammon sono impossibili (ogni campo ha già fatto uscire almeno 4 pedine): non è un'approssimazione. Il verdetto (nessun raddoppio / raddoppio, prende / raddoppio, passa) è ricostruito esattamente dalle equità memorizzate, secondo la regola di GNUbg, validata punto per punto rispetto alla sua analisi.</p>
<div class="admonition note">
<p>Le equità cubeful presuppongono un <strong>gioco di cubo ottimale di entrambi i campi fino alla fine</strong>: i recube futuri sono integralmente valorizzati (analisi retrograda completa). Nelle corse molto volatili di fine partita, la cascata di recube consuma quasi tutto il vantaggio del campo al tiro — le equità « senza raddoppio » e « raddoppio/prende » possono allora essere vicine allo zero là dove un motore come XG, il cui modello di cubo non valorizza questa cascata, mostra valori vicini al dead cube (per esempio 2 pedine sulla punta 3 contro 2 pedine sulla punta 2: 62 % di vittoria, D/T esatto +0,006 contro +0,475 per XG). La <strong>decisione</strong> mostrata, invece, coincide con quella dei motori.</p>
</div>
<p><strong>Probabilità di vittoria e verdetto, regime valutato.</strong> Fuori dal dominio esatto, la probabilità di vittoria proviene dall'output grezzo di gammonNet (ricerca a 0 ply, poi alla profondità configurata, mai letta in una tabella), e il verdetto da un «Decide» Janowski applicato a questo output — la ricerca <em>gioca</em> la traiettoria anziché riassumerne un'istantanea, ed è proprio ciò che il regime stimato non poteva fare (vedi più sotto) e permette, unico dei tre regimi insieme all'esatto, un verdetto <strong>al punteggio di match</strong>.</p>
<p>Questo regime è stato misurato, non soltanto supposto, rispetto alla tabella two-sided integrata (<code>TestEvalMeasure</code>, 4000 decisioni money campionate, parametri canonici 2 ply k=12): accordo del verdetto money <strong>93,4 %</strong> (3735/4000), ripartito per distanza dal punto di presa di gammonNet — 61,1 % a meno dell'1 % dal punto di presa (la zona più sensibile a un testa o croce), 88,3 % tra l'1 e il 5 %, 91,5 % tra il 5 e il 10 %, 94,0 % tra il 10 e il 20 %, 94,4 % oltre. Scarto di probabilità di vittoria: media 0,85 %, mediana 0,44 %, 95º percentile 3,21 %, massimo 8,30 %. Scarto di equità cubeful: media 0,039, mediana 0,018, 95º percentile 0,151, massimo 0,406. La forma è quella attesa: l'essenziale del disaccordo si concentra esattamente al punto di presa, dove due metodi legittimamente diversi divergono di più su una decisione serrata — non un errore diffuso che costerebbe equità ovunque.</p>
<p>Questa misura riguarda decisioni <strong>money</strong>, in corsa. Il verdetto al punteggio di match — che solo questo regime sa rendere — e le posizioni di contatto non hanno una misura pubblicata: quanto precede non si trasferisce a questi casi.</p>
<p><strong>Perché non più profondo di 2 ply?</strong> Perché la misura dice che non rende nulla. Una decisione di pedine costa 99 ms a 2 ply e 8,4 s a 3 ply sulla stessa macchina — <strong>ottantacinque volte di più</strong>. Su quaranta decisioni reali rigiocate a entrambe le profondità, la ricerca più profonda ha cambiato idea <strong>due volte</strong>, e in entrambi i casi il guadagno che si attribuiva valeva al massimo 0,0005 di equità normalizzata: due ordini di grandezza sotto 0,020, la soglia a partire dalla quale eXtreme Gammon parla di errore. Per decisione, tutti i casi insieme, il guadagno è 0,0000.</p>
<p>I 2 ply restano quindi l'impostazione predefinita; 3 e 4 ply si scelgono nella scheda <em>gammonNet</em> della configurazione. Non si tratta di dire che 3 ply non valga nulla in generale, ma che su <em>questa</em> rete, con il filtro canonico, non vale l'attesa di chi sta davanti a un pannello. La misura è riproducibile (<code>TestThreePlyMeasure</code>) e la conclusione sarà rigiudicata se la rete cambia.</p>
<p><strong>Perché il verdetto stimato non esiste?</strong> Quanto segue riguarda specificamente il metodo per <em>convoluzione</em> (regime stimato), non il regime valutato descritto sopra: l'equità cubeful è un problema di <em>traiettoria</em> (quando raddoppiare), che nessun riassunto statistico della posizione riesce a catturare — il miglior modello statico misurato lascia un errore residuo (deviazione standard 0,016 di equità, massimo 0,20) sufficiente a invertire tutte le decisioni serrate. Allo stesso modo, la conversione del verdetto al punteggio del match tramite una tabella di equità di match è risultata insufficiente (12 % di disaccordi con l'analisi 2-ply di GNUbg, con veri blunder). Poiché un verdetto sbagliato mostrato con sicurezza è peggio di nessun verdetto, la convoluzione non ha mai avuto il diritto di mostrare un verdetto — è una ricerca che gioca la traiettoria, non un riassunto statistico, a colmare questa lacuna.</p>
<div class="admonition note">
<p>Le basi di bearoff sono tabelle matematiche immutabili. blunderDB le calcola da sé, identiche allo strumento <code>makebearoff</code> di GNUbg — byte per byte — nella scheda <em>Bearoff</em> della configurazione o con <code>blunderdb bearoff generate</code>.</p>
</div>
<h3>Pannello Anki</h3>
<p>Il pannello <strong>Anki</strong> (<em>CTRL-K</em>) permette di studiare le posizioni tramite ripetizione dilazionata utilizzando l'algoritmo FSRS. L'utente può creare mazzi a partire da raccolte o risultati di ricerca.</p>
<p><strong>Creazione di mazzi:</strong> Fare clic su <strong>+ Nuovo mazzo</strong> per creare un mazzo a partire da una collezione o dai risultati di ricerca correnti. I mazzi basati su una ricerca si sincronizzano automaticamente all'attivazione della scheda Anki.</p>
<p><strong>Un mazzo di schede di punteggio.</strong> La terza sorgente, <em>Schede di punteggio</em>, non chiede altro che un nome: blunderDB riempie il mazzo con i 36 punteggi non ordinati da 2 a 9 away, e la carta di un punteggio è la scheda che l'esercizio Punteggi mostra — punti di presa e valori di gammon, entrambe le facce. Questo mazzo esiste solo se lo si crea: 36 carte in scadenza il primo giorno sono un debito di ripasso, e lo si contrae di proposito. Il pulsante di sincronizzazione lo rigenera.</p>
<p>I due posti non fanno lo stesso lavoro. L'esercizio Punteggi fa ritrovare quei numeri <strong>sotto l'orologio</strong> e ne misura la velocità; il mazzo li fa <strong>durare nel tempo</strong> e non ne misura nulla. Le due storie restano separate: il registro dell'Allenamento ignora i ripassi di Anki, e le statistiche di Anki ignorano le sessioni di Allenamento.</p>
<p><strong>Ripasso:</strong> Selezionare un mazzo poi fare clic su <em>Study</em> (o fare doppio clic su un mazzo) per iniziare il ripasso delle carte in scadenza. Una carta di posizione mostra la posizione sul tavoliere; una carta di punteggio annuncia il punteggio e lascia il tavoliere com'è. Valutate il vostro ricordo con i tasti <em>1</em> (Da rivedere), <em>2</em> (Difficile), <em>3</em> (Corretto), o <em>4</em> (Facile). Premere <em>Esc</em> per fermarsi e tornare all'elenco dei mazzi.</p>
<p>Due conteggi portano nomi distinti: la colonna <strong>In scadenza</strong> dell'elenco conta tutte le carte la cui scadenza è passata, comprese le carte sospese o sepolte; il numero del pulsante <em>Study</em> conta solo quelle disponibili adesso, e può quindi essere più piccolo.</p>
<p><strong>Le decisioni di cubo fanno due carte, concatenate.</strong> Una decisione di cubo è due domande — «raddoppio?», poi «presa?» — e blunderDB le registra da sempre come due posizioni. Un mazzo che ne seleziona una sola metà riceve l'altra: la decisione è completata, non ampliata. E quando entrambe sono dovute, la seconda arriva <strong>immediatamente</strong> dopo la prima.</p>
<p>Ciascuna conserva il proprio voto e il proprio calendario: non sono due tempi di una stessa carta, sono due carte. La concatenazione non anticipa alcuna scadenza — ordina le carte già dovute, nulla di più. Nascendo insieme, sono dovute insieme la prima volta, ed è lì che serve.</p>
<p><strong>Mostrare la risposta:</strong> La carta pone una domanda — quale mossa giocare, o quale azione di cubo. Riflettere, poi premere <em>SPAZIO</em> (o cliccare sulla zona nascosta) per svelare la risposta: l'analisi registrata della posizione, così come la presenta la scheda Analisi. Appare sotto i pulsanti di valutazione, che restano al loro posto e a portata di mano. Cliccare su una mossa dell'elenco la mostra sul tavoliere.</p>
<p>Nulla vi obbliga a svelare la risposta per valutare: se siete sicuri di voi, i tasti da <em>1</em> a <em>4</em> restano attivi. La risposta si rimaschera alla carta successiva, ma non se cambiate semplicemente scheda — andate a consultare il pannello Eval o il commento della posizione, vi aspetterà al ritorno.</p>
<p>Una posizione priva di analisi registrata lo indica direttamente, senza zona nascosta.</p>
<p><strong>Rispondere sul tavoliere.</strong> Per impostazione predefinita, vi valutate da soli. Nelle Impostazioni di un mazzo di posizioni, spuntate <em>Rispondere sul tavoliere</em>: per una carta di pedine, giocate allora la mossa sul tavoliere, come nell'esercizio Decisione, poi <em>Convalida</em>. Il motore giudica la mossa rispetto all'analisi registrata, svela la risposta e <strong>propone una valutazione</strong>: <em>Facile</em> per una buona risposta rapida, <em>Corretto</em> per una buona risposta più lenta, <em>Difficile</em> per un errore sotto la soglia del blunder, <em>Da rivedere</em> per un blunder o una mossa illegale. La valutazione proposta è evidenziata; mantenete il controllo e valutate ciò che volete con <em>1</em> a <em>4</em>. Una mossa legale che l'analisi non classifica non propone nulla. Le carte di cubo, le carte di punteggio e i mazzi di schede di punteggio restano in autovalutazione. Svelare la risposta senza giocare abbandona la mossa.</p>
<p><strong>Limitare la sessione.</strong> Per impostazione predefinita una sessione di ripasso arriva fino in fondo alle carte in scadenza. Puoi limitarla a un numero di carte, per mazzo, nelle Impostazioni: spunta <em>Limita la sessione</em> e indica quante carte deve servire una sessione. Quando il limite è raggiunto, la sessione si ferma dicendolo — il messaggio distingue «limite raggiunto, ancora tante carte in scadenza» da una coda davvero esaurita. Per continuare comunque c'è l'allenamento libero: propone altre posizioni senza modificare nulla della pianificazione.</p>
<p>Un limite di <strong>0</strong> non serve alcuna carta: è uno stato a sé, utile per congelare un mazzo il tempo di preparare un torneo, e non è la stessa cosa di «nessun limite». Il pulsante <em>Study</em> è allora inattivo.</p>
<p>Il limite riguarda la <strong>sessione</strong>, non la giornata. Un mazzo di blunderDB è costruito su una raccolta o su una ricerca: è un corpus finito, introdotto in poche sessioni, il cui volume quotidiano è già limitato dalla sua dimensione. Un tetto giornaliero non morderebbe mai, oppure creerebbe un arretrato su un mazzo che stava in una sessione.</p>
<p><strong>Allenamento libero (cram):</strong> Il pulsante <em>Allenamento</em>, accanto a <em>Study</em>, avvia una sessione di allenamento libero: posizioni casuali del mazzo vi vengono presentate senza tener conto della pianificazione FSRS. Questa modalità <strong>non modifica mai il piano di ripasso dilazionato</strong> — ideale per scaldarsi prima di un torneo o ripassare intensamente un mazzo tematico senza disturbarne la pianificazione. Una pastiglia <em>Libero</em> sostituisce lo stato della carta e un pulsante <em>Avanti</em> (tasti da <em>1</em> a <em>4</em>) fa scorrere le posizioni. <em>Esc</em> torna all'elenco senza registrare una sessione interrotta.</p>
<p><strong>Mettere da parte una carta, senza votarla.</strong> Durante un ripasso, un clic destro sull'intestazione della carta offre tre gesti che la tolgono dalla sessione senza dire nulla allo scheduler:</p>
<ul>
<li><strong>Sospendere</strong> — la carta conserva il suo calendario e non risale finché è sospesa. È il modo di mettere da parte una carta sbagliata, o non ancora utile, senza perdere lo storico che vi è legato.</li>
<li><strong>Rinviare</strong> — la carta sparisce fino all'indomani. A differenza della sospensione, questo non dice nulla sul suo valore: è per quella appena vista altrove, o che si preferisce non incrociare due volte in una serata.</li>
<li><strong>Togliere</strong> — la carta lascia il mazzo, previa conferma. La posizione resta nella base: un mazzo è una lista di studio sulla biblioteca, mai una sua copia.</li>
</ul>
<p>Nessuno di questi tre gesti registra un voto: una carta messa da parte non è una carta risposta, e non conta nel totale della sessione.</p>
<p><strong>Registro dei ripassi.</strong> Nelle Impostazioni di un mazzo, il pulsante <em>Registro dei ripassi</em> mostra ciò che è stato <strong>detto</strong> allo scheduler — data, posizione, voto, stato, intervallo concesso — in contrasto con ciò che prevede. È l'unico posto dove si vede un voto inserito per errore. Lì non si corregge: il calendario resta fuori portata, e proprio questa regola rende utile il registro — il passato non si riscrive, ma si può conoscere.</p>
<p><strong>Arresto/Ripresa:</strong> Potete interrompere una sessione di ripasso in qualsiasi momento con <em>Esc</em>. Il pulsante diventa <em>Riprendi</em> e mostra i vostri progressi. Fate clic su di esso per riprendere da dove vi siete fermati.</p>
<p><strong>Gestione dei mazzi:</strong> Usare i pulsanti d'azione per rinominare, sincronizzare, reimpostare o eliminare i mazzi (viene chiesta conferma per queste ultime due azioni). I parametri FSRS (ritenzione obiettivo, intervallo massimo, casualità) si possono configurare per mazzo nelle Impostazioni (icona a ingranaggio).</p>
<p><strong>Ritenzione: l'obiettivo e la misura.</strong> La <em>ritenzione obiettivo</em> è la tua scelta sul compromesso fra carico di lavoro e qualità del richiamo: più è alta, più gli intervalli si accorciano e più ripassi. Accanto, le Impostazioni mostrano la <strong>ritenzione misurata</strong> sui tuoi stessi ripassi — un'informazione, mai un comando: blunderDB non modifica il tuo obiettivo per inseguire il tuo tasso di successo. Sotto una ventina di ripassi la misura non viene mostrata: si leggerebbe come un fatto mentre è solo rumore.</p>
<p>Cambiare la ritenzione <strong>non è retroattivo</strong>: ogni carta adotta il nuovo ritmo al ripasso successivo, e le scadenze già fissate non si spostano. L'effetto è quindi graduale, e invisibile il giorno stesso.</p>
<p>L'<em>intervallo massimo</em> limita la spaziatura. Un mazzo creato di recente parte da un anno: una posizione che l'algoritmo rimanderebbe di diversi anni ha lasciato il mazzo senza che tu l'abbia deciso, e il tuo stesso gioco cambia più in fretta di così. I mazzi più vecchi conservano il valore che avevano.</p>
<h3>Pannello Allenamento</h3>
<p>Il pannello <strong>Anki</strong> fa ripassare ciò che si <strong>ricorda</strong>; il pannello <strong>Allenamento</strong> fa lavorare ciò che si <strong>calcola</strong>, sotto l'orologio. Si apre con <code>CTRL-J</code>, con il pulsante della barra degli strumenti posto subito dopo « Position aléatoire », oppure con il comando <code>train</code>. La scheda di punteggio appartiene a entrambi: si calcola qui e si ricorda in un mazzo di schede di punteggio.</p>
<p>A riposo, il pannello mostra l'avvio e il bilancio delle sessioni passate.</p>
<h4>L'avvio</h4>
<p>Tre scelte, poi « Démarrer »:</p>
<ul>
<li>l'<strong>esercizio</strong> — <em>Punteggi</em>, <em>Conteggio dei pip</em>, <em>Bearoff</em>, <em>Valutazione</em> o <em>Decisione</em>;</li>
<li>la <strong>sorgente</strong> della domanda, quando l'esercizio ne ha più d'una — <em>Vivier</em> (forme canoniche dell'esercizio), <em>Plateau</em> (la posizione così com'è) o <em>Base</em> (una posizione della lista sfogliata);</li>
<li>il <strong>limite per domanda</strong> — nessuno, 15, 30 o 60 secondi.</li>
</ul>
<p>La sorgente scelta è ricordata per ogni esercizio, da una sessione all'altra.</p>
<p>L'elenco sfogliato può provenire dagli <em>errori ricorrenti</em> del pannello Stats: un clic su «Quiz su questo gruppo» lo sostituisce con le posizioni del gruppo e avvia l'esercizio Decisione.</p>
<p><code>train scores</code>, <code>train pips</code>, <code>train bearoff</code>, <code>train evaluation</code> e <code>train decision</code> aprono il pannello e avviano direttamente; <code>train tp</code> e <code>train takepoint</code> sono sinonimi di <code>train scores</code>, <code>train epc</code> di <code>train bearoff</code>, <code>train quiz</code> di <code>train decision</code>.</p>
<h4>I cinque esercizi</h4>
<p><strong>Scores</strong> estrae a sorte uno dei 36 punteggi non ordinati da 2 a 9 away e mostra una <strong>scheda di punteggio</strong>: due colonne — <em>Vous</em> (lei) e <em>L'adversaire</em> (l'avversario) — e sette righe — il punto di presa al cubo 2 e poi al cubo 4, ciascuno in corsa lunga e all'ultimo lancio, poi il valore del gammon ai cubi 1, 2 e 4.</p>
<p>Una riga di istruzioni ricorda il gesto — stimare ogni numero a mente, poi <em>Rivelare</em>, poi premere quelli sbagliati — e la scacchiera mostra il punteggio estratto su un tavolo vuoto.</p>
<p>Ogni colonna porta soltanto le caselle che le tabelle di riferimento — quelle che mostrano i comandi <code>tp2_live</code>, <code>tp2_last</code>, <code>tp4_live</code>, <code>tp4_last</code>, <code>gv1</code>, <code>gv2</code> e <code>gv4</code> — definiscono per la sua faccia: tre numeri a 2a-2a, quattordici al massimo, e una sola colonna a punteggio pari. Una riga che nessuna delle due facce definisce non compare sulla scheda, quindi non c'è alcuna casella « n/a » da indovinare. Le due facce ci sono perché una decisione di cubo al punteggio ha bisogno di entrambe: il punto di presa corretto combina i valori di gammon dei due giocatori, ed è il punto di presa dell'avversario a dire se il suo raddoppio passa.</p>
<p><strong>Conteggio dei pip</strong> chiede il conteggio delle pedine dei <strong>due</strong> lati. Il pipcount del tavoliere è nascosto finché la domanda è aperta; «Rivela» lo mostra — <strong>anche se avevate nascosto il pipcount</strong> con <code>p</code>, altrimenti la risposta resterebbe invisibile e l'esercizio non verificabile. È una maschera e non un'impostazione: la vostra scelta non viene modificata, e riprende il controllo dalla domanda successiva. La fonte <em>Tavoliere</em> pone una domanda sulla posizione visualizzata, e una sola; la fonte <em>Database</em> estrae una nuova posizione a ogni domanda e la porta sul tavoliere.</p>
<p><strong>Bearoff</strong> chiede l'<strong>EPC dei due lati</strong> — il conteggio effettivo, quello che aggiunge al pipcount lo spreco delle pedine che escono con punti in eccesso. È il dominio in cui il motore è esatto, e quello in cui l'EPC si distingue davvero dal conteggio delle pedine.</p>
<p>Ogni domanda è <strong>generata</strong>: il motore parte da un seme e gioca qualche lancio, e l'istantanea è ciò che vi viene posto. Un piazzamento a caso non avrebbe i buchi, le pile basse e le asimmetrie di un vero bearoff. Il seme viene dal <em>Vivier</em> (un rientro completato, poi da zero a dieci mezze mosse), dal <em>Plateau</em> (la posizione mostrata, poi da una a quattro mezze mosse — mai zero, dato che l'avete appena vista) o dalla <em>Base</em> (una posizione della lista sfogliata, così com'è: è già reale).</p>
<p>Il dominio dell'esercizio: <strong>entrambi i lati interamente nella loro casa</strong>, da <strong>4 a 15 pedine</strong> per lato e le altre uscite, cubo al centro, partita a soldi. Un seme che non vi rientra è <strong>rifiutato dicendolo</strong>, e non si avvia nulla — nessun adattamento silenzioso: continuare a giocare finché il contatto si rompe vi darebbe una posizione che non avete scelto. Una tavola vuota fa eccezione: la domanda viene allora dal vivaio, e il pannello dice perché.</p>
<p>L'esercizio ha bisogno della tabella di bearoff a un lato; finché viene generata in secondo piano (vedi Configurazione), lo dice invece di porre una domanda senza risposta.</p>
<p><strong>Évaluation</strong> chiede quanto vale una posizione: la <strong>probabilità di vittoria del giocatore di turno</strong>, in percentuale, e l'<strong>azione di cubo</strong> — <em>Pas de double</em>, <em>Double, prend</em> o <em>Double, passe</em>. Sono i due numeri che mostra il pannello Eval, chiesti prima di essere mostrati. Il dominio è <strong>qualsiasi posizione</strong>: una corsa come una posizione di contatto, in <strong>partita a soldi</strong>.</p>
<p>Come per Bearoff, la domanda è <strong>generata</strong>: il seme viene dal <em>Vivier</em> (una posizione in cui il contatto si è appena rotto, poi da zero a dieci mezze mosse giocate dal motore), dal <em>Plateau</em> (la posizione visualizzata, poi da una a quattro mezze mosse; la domanda si pone in partita a soldi, con il cubo al centro, qualunque sia il punteggio del seme) o dalla <em>Base</em> (una posizione dell'elenco sfogliato, così com'è). Una posizione della base va bene solo se è una decisione di cubo in partita a soldi — senza dadi, con il cubo al centro o al giocatore di turno —; altrimenti l'estrazione passa alla successiva, e quando nessuna va bene, l'esercizio lo dice. Una tavola che non è una posizione di partita — non quindici pedine per parte, o una partita finita — <strong>viene rifiutata dicendolo</strong>; con una tavola vuota la domanda viene dal vivaio.</p>
<p>La verità è quella del motore: la base di bearoff a due parti quando la posizione vi figura, gammonNet alla sua profondità canonica ovunque altrove, e il pannello dice quale ha risposto. Nulla è stimato: una posizione che il motore non valuta non viene proposta.</p>
<p><strong>Décision</strong> pone una decisione <strong>già analizzata</strong>: una posizione della lista sfogliata — la sua sola fonte — con la decisione che porta, mossa di pedine o azione di cubo, e giudica l'analisi registrata. Una posizione senza analisi non pone domande, e una posizione già posta non torna nella sessione; quando la lista è esaurita, il pannello lo dice. Senza base aperta, o senza posizione analizzata nella lista, l'esercizio <strong>rifiuta dicendo perché</strong>, e nulla parte.</p>
<p>Finché una domanda di <em>Évaluation</em> o di <em>Décision</em> attende la sua risposta, il pannello Analisi resta mascherato: porta la risposta.</p>
<h4>Rispondere</h4>
<p>La modalità di risposta è una proprietà dell'esercizio, mai un'impostazione: ciò che si <strong>conta o si recita</strong> si dichiara, ciò che si <strong>stima</strong> si scrive — perché lì la dimensione dell'errore è la lezione.</p>
<p><em>Punteggi</em> e <em>Conteggio dei pip</em> si <strong>dichiarano</strong>: calcolate a mente, fate clic su «Rivela», e la verità viene mostrata. Ogni numero è allora <strong>giusto per impostazione predefinita</strong> — fate clic su quello che avete sbagliato per segnarlo come <strong>errore</strong> (<em>Tab</em> poi <em>Spazio</em> fa lo stesso gesto da tastiera), e un secondo clic annulla il segno. Non si digita nulla: un conteggio di pedine o una casella di tabella è giusto o sbagliato, e scriverlo non insegna niente di più che leggerlo.</p>
<p><em>Bearoff</em> si <strong>scrive</strong>: scrivete i due EPC, « Valider » li giudica a mezzo punto — la granularità alla quale l'EPC cambia una decisione di corsa — e la verità appare accanto a ciò che avete scritto. È l'applicazione che giudica, non c'è nulla da spuntare. Lo scarto è registrato <strong>con il suo segno</strong>: sopravvalutare non è sottovalutare, ed è il bilancio a farne una media.</p>
<p><em>Évaluation</em> unisce i due gesti in una stessa domanda. La probabilità di vittoria si <strong>digita</strong> e si giudica con cinque punti di margine, scarto con segno compreso; l'azione di cubo si <strong>sceglie</strong> — il clic trattiene il pulsante senza giudicare nulla, e « Valider » giudica entrambi in una volta. Il cubo non ha tolleranza: è giusto solo il pulsante che il verdetto del motore rende giusto, e a una posizione <em>troppo forte per raddoppiare</em> si risponde <em>Pas de double</em>. <em>Invio</em> nel campo porta all'azione di cubo finché non è scelta, poi convalida. Dopo la risposta, il pannello mostra la verità — il verdetto in quattro esiti —, la sua fonte, e l'<strong>EPC dei due lati</strong> quando la posizione ne ha uno esatto; l'EPC non viene mai chiesto qui, ha il suo esercizio. Il diario conta i due numeri separatamente: si può stimare bene una posizione e leggere male il suo cubo.</p>
<p><em>Décision</em> si <strong>sceglie</strong>. Su una decisione di pedine, <strong>gioca la mossa sulla tavola</strong>: clicca il punto di partenza poi la destinazione, oppure trascina la pedina, una volta per dado. La tavola offre solo ciò che è giocabile — un clic che nessuna mossa legale consente non sposta nulla. Nel pannello, « Annuler le pas » torna indietro di un dado, « Recommencer » rimette la posizione come la domanda la pone (anche il menu del clic destro propone <em>Recommencer</em>), e « Valider », attivo una volta completata la mossa, la fa giudicare. Il campo di notazione accetta anche la mossa digitata (<code>13/7 8/7</code>, la notazione della trascrizione): i passi vengono posati sulla tavola a ogni battuta, un campo arrossato segnala che un passo non è giocabile, e INVIO convalida la mossa completa. Con il focus sul pannello, BACKSPACE annulla un passo, ESC ricomincia la mossa e INVIO la convalida. Su una decisione di cubo, clicca <em>Nessun raddoppio</em>, <em>Raddoppio, presa</em> o <em>Raddoppio, passo</em>: il clic è la risposta.</p>
<p>La correzione distingue tre esiti, e confonderli mentirebbe. Una <strong>mossa illegale</strong> non è una mossa mal scelta: è un errore di regole. Una <strong>mossa legale che il motore non ha classificato</strong> non è un errore di giudizio: semplicemente non ha prezzo, e non costa nulla. Una mossa classificata costa quello che l'analisi dice, in millipunti. È giusta solo una mossa classificata e senza costo; la mossa migliore è mostrata in ogni caso.</p>
<p>Il cronometro parte alla comparsa della domanda e si ferma a « Révéler », a « Valider » o al clic di un'azione di cubo di <em>Décision</em>; la domanda successiva si prepara mentre rispondete, quindi non è mai cronometrata con la vostra. Spuntare i falli non è cronometrato nemmeno. Con un limite, una domanda rimasta senza risposta alla scadenza si rivela da sola e conta <strong>fuori tempo</strong>: tutti i suoi numeri sono sbagliati, e il suo tempo non entra nella mediana — non si misura una risposta che non è stata data. Una decisione fuori tempo mostra la mossa migliore, e non entra nel PR della sessione.</p>
<p>« Suivante » registra la domanda e ne pone un'altra. La sessione non ha una durata fissata: dura fino a « Terminer », che la scrive nel diario, o « Quitter », che la scarta. Tutti i pulsanti sono nel pannello; il tavoliere mostra la domanda e la sua risposta, non porta alcun comando.</p>
<p>Finché una domanda è posta sulla tavola, rivelata o no, i tasti che scorrono la lista non la fanno scorrere: la domanda tiene la tavola fino a « Suivante », « Terminer » o « Quitter ».</p>
<h4>Il diario e il bilancio</h4>
<p>Le sessioni terminate sono conservate nel database stesso — seguono quindi il file — e senza limite. A riposo, il pannello mostra una riga per esercizio: il numero di sessioni, il tasso di errori, il tempo mediano e, oltre dieci sessioni, la <strong>tendenza</strong>, cioè lo scarto tra il tasso di errori delle ultime dieci sessioni e quello di tutte — negativo, state migliorando.</p>
<p>Per <em>Décision</em>, la riga dà anche il <strong>PR</strong> dell'ultima sessione, calcolato con la formula che le statistiche applicano al gioco reale — 500 × errore medio in equità normalizzata, sulle decisioni giudicate. Un PR di allenamento di 6 e un PR di incontro di 6 misurano la stessa cosa sulla stessa scala.</p>
<p>Facendo clic sul nome dell'esercizio si apre il dettaglio <strong>per tipo di numero</strong>: « Point de prise 4 · dernier lancer, 6 / 9 ». È questo dettaglio a dare valore al diario, e conta per tipo e non per faccia: la stessa casella della stessa tabella, vista da un lato o dall'altro, è una sola debolezza.</p>
<p>Una domanda di <em>Decisione</em> conserva nel registro la sua posizione, la risposta data e il suo costo in millipunti. Il dettaglio di <em>Decisione</em> include quindi tre pulsanti che agiscono su tutte le posizioni sbagliate, ciascuna una volta, la sbagliata più di recente per prima — una domanda scaduta conta come sbagliata:</p>
<ul>
<li><strong>Riprendi i miei errori</strong> — le posizioni sbagliate diventano la lista scorsa e una sessione <em>Decisione</em> riparte su di esse;</li>
<li><strong>Mazzo Anki degli errori</strong> — un mazzo Anki con queste posizioni;</li>
<li><strong>Collezione degli errori</strong> — una collezione con queste posizioni.</li>
</ul>
<p>Il mazzo e la collezione si chiamano «Errori in Decisione» seguito dalla data odierna. Da riga di comando, <code>training missed</code> restituisce la stessa lista e ne fa un mazzo (<code>--deck</code>) o una collezione (<code>--collection</code>), e <code>training sessions</code> rilegge il registro (vedi training — Il registro di allenamento).</p>
<h3>Pannello Duello</h3>
<p>Il pannello <strong>Duello</strong> fa giocare un intero match contro il Bot, con blunderDB come Arbitro: tira i dadi, impone le regole e tiene il punteggio e gli orologi. Si apre con <code>CTRL-H</code>, con il pulsante «Gioca» della barra degli strumenti o con il comando <code>duel</code>. Deve essere aperta una base di dati: il Duello vi viene scritto dopo ogni decisione.</p>
<p>Senza un Duello aperto, il pannello mostra il modulo, memorizzato da un Duello all'altro, e l'elenco dei Duelli sospesi:</p>
<ul>
<li><strong>Match</strong> da 1 a 25 punti, oppure <strong>sessione money</strong> (Jacoby a scelta).</li>
<li><strong>Partenza</strong>: la posizione iniziale, la posizione sul tavoliere, oppure la posizione iniziale a un punteggio scelto.</li>
<li><strong>Lato giocato</strong> (giocatore 1 o 2) e <strong>livello del Bot</strong> (<code>instant</code>, <code>normal</code>, <code>thorough</code>, quelli dell'analisi). Il selettore indica la profondità di ogni livello, per esempio «instant (0-ply)» o «normal (2-ply, potato)»; più profondo è più forte e più lento.</li>
<li><strong>Cadenza</strong>: senza cadenza, o una cadenza con nome (una riserva per giocatore e un ritardo gratuito a ogni turno), e ciò che avviene allo scadere del tempo: continuare annotandolo, o perdere il match.</li>
<li><strong>Il tuo nome</strong> e <strong>Salva il match</strong>: se deselezionato, il Duello terminato viene scartato invece di diventare un Match.</li>
</ul>
<p>«Riprendi» riapre un Duello sospeso nello stesso punto, con gli stessi dadi a venire; i suoi orologi erano fermi.</p>
<p>Un Duello sospeso non lascia la sua base: l'esportazione della base non lo porta con sé, perché i suoi dadi a venire non devono uscire per nessuna via prima della fine. Un Duello terminato si esporta come qualsiasi Match.</p>
<p>Il Duello si gioca sulla scacchiera. Il pannello mostra il foglio del match in due colonne, come la Trascrizione, gli orologi quando corre una cadenza, una riga che dice ciò che è atteso, e «Abbandona il match», «Metti in pausa», «Annulla il match». Un match a punti si salva solo intero: nessun arresto conserva un match incompiuto. Il punteggio e il cubo sono quelli della scacchiera; il punteggio e gli orologi restano nella barra di stato quando la scheda è ripiegata. L'impronta SHA-256 del seme dei dadi, pubblicata dall'Arbitro alla creazione, si legge nel suggerimento a comparsa della riga di indicazione del pannello, e con il seme nell'origine del Match terminato. La scacchiera passa in modalità <strong>DUELLO</strong>: la libreria non si sfoglia più, la modifica, il pannello Eval e le altre schede non si aprono, e il motore tace — nessuna valutazione, nessun candidato. Restano solo la Pila (<code>B</code>), il pipcount (<code>P</code>) e la guida.</p>
<ul>
<li>Prima del lancio, un clic sulla scacchiera, sui dadi o su una pedina li lancia; solo un clic sul cubo propone di raddoppiare, e la scacchiera chiede «Raddoppia» o «Annulla». Quando il cubo non è disponibile, il lancio è automatico.</li>
<li>Di fronte a un raddoppio del Bot, la scacchiera chiede «Accetta» o «Rifiuta».</li>
<li>Un clic su una pedina la gioca con il dado di sinistra ancora libero, o con l'altro quando quello di sinistra non può giocarla; un doppio si gioca con quattro clic. Una pedina può anche essere trascinata verso la sua destinazione. Un dado giocato è attenuato. Passano solo i passi di una mossa legale.</li>
<li>Prima di giocare, un clic sui dadi, o un clic destro sulla scacchiera, ne inverte l'ordine. Durante la mossa, il clic destro sulla scacchiera riprende tutte le pedine giocate (anche <code>BACKSPACE</code>).</li>
<li>La mossa completa si convalida con un clic sui dadi, con «Convalida» sulla scacchiera, o con <code>INVIO</code> o <code>SPAZIO</code>. Dopo non si riprende nulla.</li>
<li>Il Bot risponde subito; le sue mosse vengono rigiocate sul tavoliere, lentamente.</li>
<li>Il clic destro fuori dalla scacchiera, o sulla scacchiera fuori dalla sua mossa, apre il menu del Duello: mettere la posizione sulla Pila o toglierla, abbandonare la partita per un semplice, un gammon o un backgammon (al proprio turno, dopo conferma), abbandonare il match, metterlo in pausa, annullarlo. Questo menu non offre né valutazione né modifica.</li>
<li>«Abbandona il match» cede l'intero match, in qualsiasi momento e dopo conferma: la partita in corso va all'avversario per i punti che lo portano alla lunghezza, e il Match viene salvato come vinto da lui. In money, la partita in corso è persa come semplice, al valore del cubo (con o senza regola Jacoby), e la sessione si chiude. Sotto una cadenza che fa perdere il match, una riserva esaurita vale l'abbandono del match da parte di quel giocatore.</li>
<li>«Metti in pausa» sospende il Duello, orologi fermi; riprende dallo stesso punto. «Annulla il match» lo scarta, dopo conferma: non viene salvato nulla.</li>
<li>Un doppio clic fuori dalla scacchiera mette la posizione sulla Pila, o la toglie, come <code>B</code>; solo il segnalibro all'angolo della scacchiera lo mostra.</li>
</ul>
<p>Ogni decisione porta la sua durata, con o senza cadenza. Un abbandono non ha una durata registrata nel Match.</p>
<p>Alla fine il Match viene scritto, la sua analisi parte e la scheda Partite si apre su di esso. Da riga di comando, <code>blunderdb duel</code> pilota lo stesso Duello (vedi duel — Giocare un Duello).</p>
<h3>Pannello Metadati</h3>
<p>Il pannello <strong>Metadati</strong> (<em>CTRL-M</em>) mostra le informazioni generali del database corrente: <em>Utente</em>, data di creazione (<em>Creato</em>), <em>Versione</em> dello schema e <em>Descrizione</em>. L'utente, la data e la descrizione si modificano sul posto e si salvano uscendo dal campo; la versione è in sola lettura. Accessibile anche tramite il comando <code>meta</code>.</p>
<p>Mostra inoltre, <strong>quando esiste</strong>, l'origine del database — vedere Distribuire un database: origine e password. Un database ordinario non mostra questa sezione.</p>
<h3>Distribuire un database: origine e password</h3>
<p>Un insegnante che distribuisce un database di posizioni dispone di due meccanismi, indipendenti l'uno dall'altro, entrambi facoltativi e scelti <strong>al momento dell'esportazione</strong>: contrassegnare il file con la sua origine e proteggerlo con una password.</p>
<div class="admonition note">
<p>Nessuno dei due tiene traccia di ciò che accade al file. blunderDB <strong>non registra nulla dal lato di chi riceve il database</strong>: aprire un database contrassegnato è esattamente come aprirne uno qualsiasi, e da nessuna parte viene annotato chi lo ha aperto, quando, né da dove proviene il suo contenuto.</p>
</div>
<h4>Contrassegnare un database con la sua origine</h4>
<p>La finestra di esportazione sta in una sola schermata: il modulo e, sovrapposto ad esso durante la scrittura, un indicatore di avanzamento. Si chiude da sola al termine e il risultato appare nella barra di stato.</p>
<p>Tre punti meritano attenzione:</p>
<ul>
<li><strong>L'esportazione riguarda le posizioni attualmente visualizzate</strong>, non l'intero database. Dopo una ricerca vengono esportati solo i risultati: la finestra lo ricorda in alto.</li>
<li><strong>Una raccolta le cui posizioni non siano tutte nella selezione arriva troncata.</strong> L'elenco mostra quindi, per ogni raccolta, la parte coperta («12/40») e la segnala in rosso quando è parziale.</li>
<li><strong>I tornei possono essere esportati solo insieme ai match</strong>: senza di essi il collegamento torneo–match non esiste e il torneo arriverebbe vuoto. La casella resta disattivata finché non è selezionato «includi i match».</li>
</ul>
<p>I campi <em>Utente</em>, <em>Descrizione</em> e <em>Data di creazione</em> descrivono il <strong>file prodotto</strong>; sono precompilati dal database sorgente. La casella <em>I miei filtri salvati</em> è a parte rispetto alle altre: non esporta contenuto ma le vostre ricerche salvate, inutili nel database di qualcun altro.</p>
<p>Selezionando <strong>Contrassegna questo file con la sua origine</strong> compaiono due campi:</p>
<ul>
<li><strong>Origine</strong> — che cos'è questo file e da dove proviene, con parole tue: «Lezione di Jean Dupont — 12 marzo 2026». Questo campo è <strong>obbligatorio</strong>: finché è vuoto, il pulsante di esportazione resta inattivo.</li>
<li><strong>Nota</strong>, facoltativa — condizioni d'uso, indirizzo di contatto, la richiesta di non ridistribuire.</li>
</ul>
<p>Il contrassegno è firmato con la tua identità di emittente. È quindi <strong>inalterabile e non falsificabile</strong>: nessuno può modificarlo né fabbricarne uno a tuo nome. Non è invece <strong>incancellabile</strong>: il file distribuito è un normale database SQLite e blunderDB è software libero. Non impedisce nulla: dice da dove viene il file.</p>
<h4>Identità dell'emittente</h4>
<p>I contrassegni sono firmati con la tua <strong>identità di emittente</strong>, creata da sé la prima volta che contrassegni un file; non c'è nulla da configurare. Appartiene a una persona e non a un database: tutti i tuoi file recano la stessa impronta pubblica, nella forma <code>A3F1-9C24-7B05-E1D8</code>.</p>
<p>Puoi comunicare questa impronta ai tuoi destinatari perché verifichino che un file proviene davvero da te. L'identità si sposta da un computer all'altro in un unico file (estensione <code>.bdbid</code>), eventualmente protetto da una passphrase. <strong>Questo file permette di firmare a tuo nome: non condividerlo.</strong></p>
<p>Nelle impostazioni (icona a ingranaggio della barra degli strumenti), la scheda <em>Identità dell'emittente</em> mostra il tuo nome e la tua impronta e propone <em>Salva identità…</em>, <em>Carica identità…</em> e <em>Rigenera…</em>.</p>
<div class="admonition warning">
<p><strong>Rigenerare non revoca nulla.</strong> Un contrassegno incorpora la chiave pubblica che lo ha firmato: si verifica quindi per sempre, da sé. Se il tuo file di identità è trapelato, chi lo possiede potrà continuare a firmare con la tua vecchia impronta, e quei contrassegni resteranno validi.</p>
<p>Ciò che ti protegge dopo una fuga non è il software: è pubblicare la tua nuova impronta e disconoscere la vecchia presso i tuoi destinatari.</p>
<p>La rigenerazione sovrascrive la chiave attuale; blunderDB propone di salvarla prima di sostituirla.</p>
</div>
<h4>Proteggere un database con una password</h4>
<p>La password si digita mascherata, qui come all'apertura di un file protetto; l'icona a forma di occhio la mostra <strong>finché la si tiene premuta</strong> e la nasconde di nuovo appena la si rilascia.</p>
<p>Selezionando <strong>Proteggi questo file con una password</strong> si ottiene un file con estensione <code>.dbx</code>, anche se nella finestra di salvataggio avevi scelto un nome in <code>.db</code>, poiché tale finestra si apre prima che venga chiesta la password. Per aprirlo, usa la consueta apertura di un database: la finestra di selezione accetta sia i <code>.db</code> sia i <code>.dbx</code>. blunderDB chiede allora la password e installa accanto un database ordinario; in seguito non viene più chiesto nulla.</p>
<p>La finestra propone di <strong>eliminare il file protetto una volta aperto</strong>: altrimenti si conserva lo stesso contenuto sotto due nomi. La casella non è selezionata per impostazione predefinita — il file protetto resta tuo se intendi trasmetterlo — e l'eliminazione avviene solo dopo un'apertura riuscita.</p>
<div class="admonition warning">
<p>La password protegge il <strong>trasporto</strong> del file, non il database. Impedisce a un terzo di aprire un file dimenticato in una cartella di download o un allegato inoltrato per errore. Non protegge da colui al quale hai dato la password.</p>
</div>
<p>La password viene verificata a <strong>ogni</strong> apertura, anche quando il file è già stato aperto in precedenza su quel computer.</p>
<p>Tecnicamente il database è cifrato con <strong>AES-256 in modalità GCM</strong>, con una chiave derivata dalla password tramite <strong>Argon2id</strong> (64 MiB di memoria, 3 passaggi, 4 thread) e un sale casuale proprio di ogni file. La modalità GCM autentica l'insieme: una password errata viene rilevata come tale, e così pure qualsiasi alterazione del file cifrato — non si ottiene mai in silenzio un database corrotto.</p>
<p>L'intestazione del file protetto resta <strong>in chiaro</strong>: la sua origine rimane leggibile senza la password.</p>
<h4>Leggere l'origine di un file</h4>
<p>Nell'applicazione, apri il file e mostra il pannello <strong>Metadati</strong> (comando <code>meta</code>). In cima al pannello compare una sezione <strong>Origine</strong>, in sola lettura, che indica ciò che è stato iscritto, da chi, quando, e lo stato della firma:</p>
<ul>
<li>«✓ marcato da voi»: il file porta il vostro marchio, intatto;</li>
<li>«✓ firma verificata»: il contrassegno è intatto e proviene da un'altra chiave — confronta la sua impronta con quella che l'autore ti ha comunicato;</li>
<li>«⚠ firma non valida»: il documento è stato modificato o contraffatto.</li>
</ul>
<p>Questa sezione non compare in un database ordinario.</p>
<p>Da riga di comando, <code>blunderdb info --db file.db</code> mostra l'origine e lo stato della firma, <strong>senza mai scrivere nel file</strong>. Il comando funziona anche su un file protetto, senza la password. Vedere <code>CLI_USAGE.md</code> per le opzioni <code>--watermark</code> e <code>--password</code> di <code>export</code>, nonché per <code>identity</code> e <code>open</code>.</p>
<h4>Pubblicare una base per altri</h4>
<p>Una base marcata si distribuisce come qualsiasi file — email, sito personale, chiavetta USB. blunderDB <strong>non fornisce alcun servizio</strong>: né deposito, né catalogo ospitato, né account. È una conseguenza diretta della sua concezione: dal lato di chi riceve un file non viene mai registrato nulla, e non ci sarebbe quindi nulla da comunicare a un servizio, anche se esistesse.</p>
<p>Ciò che rende una base pubblicata utilizzabile da un altro si riduce a quattro campi, tutti già presenti:</p>
<ul>
<li><strong>Utente</strong> — chi l'ha costituita, con il nome che vuoi veder citato.</li>
<li><strong>Descrizione</strong> — che cosa contiene la base, in una frase che stia in un elenco: «240 decisioni di cubo al punteggio, commentate, livello intermedio».</li>
<li><strong>Provenienza</strong> (della filigrana) — che cos'è questo file e per chi è stato prodotto. È la prima cosa che il destinatario legge nel pannello <em>Metadati</em>.</li>
<li><strong>Impronta dell'emittente</strong> — pubblicala accanto al file, non dentro: è confrontandola che il destinatario verifica che il file viene da te e non da qualcuno che ha ripreso il tuo nome.</li>
</ul>
<p>Una base pubblicata senza filigrana resta perfettamente utilizzabile; è semplicemente anonima, e il pannello <em>Metadati</em> non mostra allora alcuna sezione <em>Provenienza</em>.</p>
<p>Per far conoscere una base, la categoria <em>Show and tell</em> delle <code>discussioni del deposito &lt;https://github.com/kevung/blunderDB/discussions&gt;</code>_ fa da elenco: è una lista tenuta da chi pubblica, non un servizio reso da blunderDB. Annunciarne una lì richiede il link, i quattro campi qui sopra e l'impronta.</p>
`,
    shortcuts: `
<p>I suggerimenti della barra degli strumenti ricordano il tasto di ogni pulsante con il nome che ha sulla tastiera della lingua dell'interfaccia: <em>Sinistra</em>, <em>Canc</em>, <em>Pag su</em>, <em>Pag giù</em>, <em>Maiusc</em>. Le tabelle seguenti e la finestra di aiuto usano gli stessi nomi.</p>
<h3>Database</h3>
<table>
<thead>
<tr>
<th>Scorciatoia</th>
<th>Azione</th>
</tr>
</thead>
<tbody>
<tr>
<td>CTRL-N</td>
<td>Crea un nuovo database.</td>
</tr>
<tr>
<td>CTRL-O</td>
<td>Apri un database esistente.</td>
</tr>
<tr>
<td>CTRL-MAIUSC-I</td>
<td>Unire un database a questo.</td>
</tr>
<tr>
<td>CTRL-MAIUSC-S</td>
<td>Esporta il database.</td>
</tr>
<tr>
<td>CTRL-Q</td>
<td>Chiudi blunderDB.</td>
</tr>
<tr>
<td>CTRL-M</td>
<td>Modifica i metadati del database.</td>
</tr>
</tbody>
</table>
<h3>Posizione</h3>
<table>
<thead>
<tr>
<th>Scorciatoia</th>
<th>Azione</th>
</tr>
</thead>
<tbody>
<tr>
<td>CTRL-I</td>
<td>Importa una o più posizioni/partite da file (xg, xgp, sgf, mat, txt, bgf).</td>
</tr>
<tr>
<td>CTRL-MAIUSC-F</td>
<td>Importa ricorsivamente una cartella di file di partite/posizioni.</td>
</tr>
<tr>
<td>CTRL-C</td>
<td>Copia una posizione negli appunti.</td>
</tr>
<tr>
<td>CTRL-X</td>
<td>Copia l'immagine del board negli appunti (PNG).</td>
</tr>
<tr>
<td>CTRL-X CTRL-X</td>
<td>Copia l'immagine del board con l'analisi negli appunti (PNG).</td>
</tr>
<tr>
<td>CTRL-V</td>
<td>Incolla una posizione dagli appunti (rilevamento automatico del formato).</td>
</tr>
<tr>
<td>CTRL-S</td>
<td>Salva una posizione.</td>
</tr>
<tr>
<td>CTRL-U</td>
<td>Aggiorna una posizione.</td>
</tr>
<tr>
<td>Canc</td>
<td>Elimina la posizione corrente (viene chiesta conferma).</td>
</tr>
<tr>
<td>BACKSPACE</td>
<td>In modalità modifica o Eval: reimpostare il tavoliere, il cubo, il punteggio e i dadi.</td>
</tr>
<tr>
<td>CTRL-G</td>
<td>Mostra i metadati della posizione.</td>
</tr>
<tr>
<td>b</td>
<td>Mettere la posizione mostrata nella Pila (la raccolta «da rivedere più tardi») o toglierla.</td>
</tr>
<tr>
<td>Doppio clic fuori dalla scacchiera</td>
<td>Mettere la posizione mostrata sulla Pila, o toglierla, in tutte le modalità.</td>
</tr>
<tr>
<td>Clic destro fuori dalla scacchiera (modifica, Eval)</td>
<td>Aprire il menu della scacchiera: <em>Cancella la posizione</em>, come BACKSPACE, o <em>Posizione iniziale</em>.</td>
</tr>
<tr>
<td>Clic destro sulla scacchiera (mossa giocata sulla scacchiera)</td>
<td>Aprire il menu della scacchiera, che inizia con <em>Ricomincia</em>: si azzera la mossa, non la posizione.</td>
</tr>
</tbody>
</table>
<h3>Navigazione</h3>
<table>
<thead>
<tr>
<th>Scorciatoia</th>
<th>Azione</th>
</tr>
</thead>
<tbody>
<tr>
<td>CTRL-R</td>
<td>Ricarica tutte le posizioni dal database.</td>
</tr>
<tr>
<td>Home, h</td>
<td>Prima posizione / Partita precedente (navigazione match).</td>
</tr>
<tr>
<td>Pag su</td>
<td>Torna indietro di una pagina (cento posizioni per impostazione predefinita, regolabile in Impostazioni &gt; Interfaccia: 10, 50, 100, 500 o 1 000 posizioni, oppure il 10 % dell'elenco; si ferma all'inizio dell'elenco); in un match, partita precedente.</td>
</tr>
<tr>
<td>SINISTRA, k</td>
<td>Posizione precedente.</td>
</tr>
<tr>
<td>DESTRA, j</td>
<td>Posizione successiva.</td>
</tr>
<tr>
<td>SU, k</td>
<td>Mossa precedente (quando una mossa è selezionata nell'analisi).</td>
</tr>
<tr>
<td>GIÙ, j</td>
<td>Mossa successiva (quando una mossa è selezionata nell'analisi).</td>
</tr>
<tr>
<td>Fine, l</td>
<td>Ultima posizione / Partita successiva (navigazione match).</td>
</tr>
<tr>
<td>Pag giù</td>
<td>Avanza di una pagina (stesso passo di <em>Pag su</em>; si ferma alla fine dell'elenco); in un match, partita successiva.</td>
</tr>
<tr>
<td>r</td>
<td>Carica una posizione casuale.</td>
</tr>
<tr>
<td>ESC</td>
<td>Uscire dai risultati di una ricerca <code>ss</code> avviata da una collezione o da un match: ritorno alla collezione, o al match sulla mossa studiata.</td>
</tr>
</tbody>
</table>
<p>Finché una domanda del pannello Allenamento è posta sulla tavola, i tasti che scorrono la lista non la fanno scorrere: la domanda tiene la tavola. Su una decisione di pedine, con il focus sul pannello: BACKSPACE annulla l'ultimo passo, ESC ricomincia la mossa, INVIO la convalida quando è completa; nel campo di notazione, INVIO convalida la mossa digitata (<code>13/7 8/7</code>).</p>
<h3>Visualizzazione</h3>
<table>
<thead>
<tr>
<th>Scorciatoia</th>
<th>Azione</th>
</tr>
</thead>
<tbody>
<tr>
<td>CTRL-SINISTRA</td>
<td>Orientamento del board a sinistra.</td>
</tr>
<tr>
<td>CTRL-DESTRA</td>
<td>Orientamento del board a destra.</td>
</tr>
<tr>
<td>p</td>
<td>Mostra/nascondi il conteggio dei pip.</td>
</tr>
</tbody>
</table>
<h3>Azioni</h3>
<table>
<thead>
<tr>
<th>Scorciatoia</th>
<th>Azione</th>
</tr>
</thead>
<tbody>
<tr>
<td>TAB</td>
<td>Apri il pannello di ricerca (editor di posizione).</td>
</tr>
<tr>
<td>SPAZIO</td>
<td>Apri la riga di comando.</td>
</tr>
<tr>
<td>ALT-1 … ALT-9</td>
<td>Lanciare il filtro fissato di quel rango nella libreria dei filtri.</td>
</tr>
</tbody>
</table>
<h3>Strumenti</h3>
<table>
<thead>
<tr>
<th>Scorciatoia</th>
<th>Azione</th>
</tr>
</thead>
<tbody>
<tr>
<td>CTRL-L</td>
<td>Mostra/nascondi l'analisi.</td>
</tr>
<tr>
<td>CTRL-MAIUSC-L</td>
<td>Ordina le vicine della posizione corrente.</td>
</tr>
<tr>
<td>CTRL-P</td>
<td>Mostra/nascondi i commenti.</td>
</tr>
<tr>
<td>CTRL-MAIUSC-P</td>
<td>Aprire/chiudere la tavolozza dei comandi: comandi, schede, filtri e match, ritrovati con un nome approssimativo.</td>
</tr>
<tr>
<td>CTRL-J</td>
<td>Mostrare/nascondere il pannello Allenamento.</td>
</tr>
<tr>
<td>CTRL-H</td>
<td>Mostra/nascondi il pannello Duello (un match contro il Bot).</td>
</tr>
<tr>
<td>INVIO (Duello)</td>
<td>Convalida la mossa disposta sul tavoliere; dopo non si può più annullare nulla.</td>
</tr>
<tr>
<td>SPAZIO (Duello)</td>
<td>Convalidare la mossa disposta sulla scacchiera, una volta giocati tutti i dadi; nessun effetto su una mossa parziale o fuori dal vostro turno.</td>
</tr>
<tr>
<td>BACKSPACE (Duello)</td>
<td>Rimette le pedine a posto prima della convalida.</td>
</tr>
<tr>
<td>CTRL-K</td>
<td>Mostra/nascondi il pannello Anki (ripetizione dilazionata).</td>
</tr>
<tr>
<td>CTRL-F</td>
<td>Mostra/nascondi il pannello di ricerca.</td>
</tr>
<tr>
<td>CTRL-Tab</td>
<td>Mostra/nascondi il pannello dei match.</td>
</tr>
<tr>
<td>CTRL-B</td>
<td>Mostra/nascondi il pannello delle collezioni.</td>
</tr>
<tr>
<td>CTRL-Y</td>
<td>Mostra/nascondi il pannello dei tornei.</td>
</tr>
<tr>
<td>CTRL-D</td>
<td>Mostra/nascondi il pannello delle statistiche.</td>
</tr>
<tr>
<td>CTRL-E</td>
<td>Mostra/nascondi il pannello Eval.</td>
</tr>
<tr>
<td>CTRL-MAIUSC-T</td>
<td>Mostra/nascondi il pannello Trascrizione (bozze di partite).</td>
</tr>
<tr>
<td>J / K</td>
<td>Nella coda delle proposte di un torneo diretto: giù, su.</td>
</tr>
<tr>
<td>INVIO</td>
<td>Nella coda delle proposte: confermare la proposta selezionata.</td>
</tr>
<tr>
<td>Pagina del torneo (TAB)</td>
<td>Prima sosta nella scheda Direzione: «Vai alla coda»; INVIO porta il fuoco sulla coda delle proposte.</td>
</tr>
<tr>
<td>SINISTRA / DESTRA</td>
<td>Nella scheda del risultato, fuori da un campo: scegliere il giocatore di sinistra o quello di destra; INVIO registra la sua vittoria.</td>
</tr>
<tr>
<td>CTRL-Z</td>
<td>Riprendere l'ultima decisione di un torneo diretto (fuori da un campo di immissione, dove annulla la battuta).</td>
</tr>
<tr>
<td>TAB (focus perso)</td>
<td>Sotto la pagina di un torneo diretto, quando il focus è caduto sulla pagina: riportarlo sul primo elemento della pagina, senza aprire la ricerca.</td>
</tr>
<tr>
<td>PAG SU / PAG GIÙ, HOME / FINE</td>
<td>Sotto la pagina di un torneo diretto: scorrere la pagina, senza percorrere il tavoliere che nasconde.</td>
</tr>
<tr>
<td>ESC</td>
<td>Chiudere la scheda del risultato o la ripresa in corso.</td>
</tr>
<tr>
<td>?</td>
<td>Mostra/nascondi l'aiuto.</td>
</tr>
</tbody>
</table>
<h3>Duello sulla scacchiera</h3>
<table>
<thead>
<tr>
<th>Gesto</th>
<th>Azione</th>
</tr>
</thead>
<tbody>
<tr>
<td>Clic sulla scacchiera o sui dadi (prima del lancio)</td>
<td>Lanciare i dadi. Il cubo conserva il suo significato: propone di raddoppiare.</td>
</tr>
<tr>
<td>Clic sul cubo (prima del lancio)</td>
<td>Proporre di raddoppiare; «Raddoppia» o «Annulla» conferma sulla scacchiera.</td>
</tr>
<tr>
<td>Clic su una pedina</td>
<td>Giocarla con il dado di sinistra ancora libero, o con l'altro se quello di sinistra non può giocarla. Un dado giocato è attenuato.</td>
</tr>
<tr>
<td>Trascinare una pedina</td>
<td>Giocarla verso il punto in cui viene rilasciata, se una mossa legale lo permette.</td>
</tr>
<tr>
<td>Clic sui dadi (mossa in corso)</td>
<td>Nessun dado giocato: invertirne l'ordine. Mossa completa: convalidarla.</td>
</tr>
<tr>
<td>Clic destro sulla scacchiera (mossa in corso)</td>
<td>Riprendere tutte le pedine giocate; senza pedine giocate, invertire i dadi.</td>
</tr>
<tr>
<td>Clic destro fuori dalla scacchiera, o fuori dalla sua mossa</td>
<td>Aprire il menu del Duello: Pila, abbandono, sospensione, arresto.</td>
</tr>
</tbody>
</table>
<h3>Schede delle viste</h3>
<table>
<thead>
<tr>
<th>Scorciatoia</th>
<th>Azione</th>
</tr>
</thead>
<tbody>
<tr>
<td>CTRL-T</td>
<td>Crea una nuova vista (copia della vista corrente).</td>
</tr>
<tr>
<td>CTRL-W</td>
<td>Chiudi la vista corrente.</td>
</tr>
<tr>
<td>CTRL-Pag su, MAIUSC-J</td>
<td>Vista precedente.</td>
</tr>
<tr>
<td>CTRL-Pag giù, MAIUSC-K</td>
<td>Vista successiva.</td>
</tr>
<tr>
<td>CTRL-1 … CTRL-9</td>
<td>Vai direttamente all'n-esima vista.</td>
</tr>
<tr>
<td>Doppio clic sulla scheda</td>
<td>Rinomina la vista.</td>
</tr>
</tbody>
</table>
<h3>Riga di comando</h3>
<table>
<thead>
<tr>
<th>Scorciatoia</th>
<th>Azione</th>
</tr>
</thead>
<tbody>
<tr>
<td>SU</td>
<td>Scorri la cronologia dei comandi verso l'alto.</td>
</tr>
<tr>
<td>GIÙ</td>
<td>Scorri la cronologia dei comandi verso il basso.</td>
</tr>
<tr>
<td>ESC</td>
<td>Durante una ricerca: interromperla.</td>
</tr>
</tbody>
</table>
<h3>Cronologia di ricerca</h3>
<table>
<thead>
<tr>
<th>Scorciatoia</th>
<th>Azione</th>
</tr>
</thead>
<tbody>
<tr>
<td>Clic</td>
<td>Seleziona/deseleziona una ricerca (mostra la posizione).</td>
</tr>
<tr>
<td>Doppio clic</td>
<td>Esegui la ricerca.</td>
</tr>
</tbody>
</table>
<h3>Libreria di filtri</h3>
<table>
<thead>
<tr>
<th>Scorciatoia</th>
<th>Azione</th>
</tr>
</thead>
<tbody>
<tr>
<td>Clic</td>
<td>Seleziona/deseleziona un filtro (mostra la posizione).</td>
</tr>
<tr>
<td>Doppio clic</td>
<td>Esegui la ricerca del filtro.</td>
</tr>
<tr>
<td>Clic (sulla stella)</td>
<td>Fissare o sbloccare il filtro.</td>
</tr>
<tr>
<td>Clic (su un'etichetta fissata)</td>
<td>Eseguire la ricerca del filtro fissato.</td>
</tr>
</tbody>
</table>
<h3>Pannello di analisi</h3>
<table>
<thead>
<tr>
<th>Scorciatoia</th>
<th>Azione</th>
</tr>
</thead>
<tbody>
<tr>
<td>Clic</td>
<td>Seleziona/deseleziona una mossa (mostra/nascondi le frecce).</td>
</tr>
<tr>
<td>Ctrl+Clic</td>
<td>Aggiungere o togliere una mossa dalla selezione da rollare.</td>
</tr>
<tr>
<td>Maiusc+Clic</td>
<td>Estendere la selezione fino alla mossa cliccata.</td>
</tr>
<tr>
<td>Clic destro</td>
<td>Menu del rollout: rollare le mosse selezionate con l'impostazione scelta, o annullare il rollout in corso; copiare la posizione e l'analisi, o la posizione e le mosse selezionate.</td>
</tr>
<tr>
<td>SU, k</td>
<td>Seleziona la mossa precedente (quando una mossa è selezionata).</td>
</tr>
<tr>
<td>GIÙ, j</td>
<td>Seleziona la mossa successiva (quando una mossa è selezionata).</td>
</tr>
<tr>
<td>d</td>
<td>Alterna tra l'analisi delle mosse e del cubo (solo navigazione match).</td>
</tr>
<tr>
<td>r</td>
<td>Avviare il rollout delle mosse selezionate (senza selezione, della posizione) con l'impostazione scelta; una seconda pressione lo ferma.</td>
</tr>
<tr>
<td>Esc</td>
<td>Fermare il rollout in corso, altrimenti deselezionare la mossa. Se nessuna mossa è selezionata, chiudere il pannello, tranne davanti ai risultati di una ricerca <code>ss</code>: tornare da lì alla collezione o al match.</td>
</tr>
</tbody>
</table>
<h3>Pannello Eval</h3>
<table>
<thead>
<tr>
<th>Scorciatoia</th>
<th>Azione</th>
</tr>
</thead>
<tbody>
<tr>
<td>Clic</td>
<td>Seleziona/deseleziona una mossa (mostra/nascondi le frecce).</td>
</tr>
<tr>
<td>Ctrl+Clic</td>
<td>Aggiungere o togliere una mossa dalla selezione da rollare.</td>
</tr>
<tr>
<td>Maiusc+Clic</td>
<td>Estendere la selezione fino alla mossa cliccata.</td>
</tr>
<tr>
<td>Clic destro</td>
<td>Menu del rollout: rollare le mosse selezionate con l'impostazione scelta, o annullare il rollout in corso; copiare la posizione e l'analisi, o la posizione e le mosse selezionate.</td>
</tr>
<tr>
<td>SU, k</td>
<td>Seleziona la mossa precedente (quando una mossa è selezionata).</td>
</tr>
<tr>
<td>GIÙ, j</td>
<td>Seleziona la mossa successiva (quando una mossa è selezionata).</td>
</tr>
<tr>
<td>r</td>
<td>Avviare il rollout delle mosse selezionate (senza selezione, della posizione) con l'impostazione scelta; una seconda pressione lo ferma.</td>
</tr>
<tr>
<td>Esc</td>
<td>Fermare il rollout in corso, altrimenti deselezionare la mossa.</td>
</tr>
</tbody>
</table>
<h3>Pannello dei match</h3>
<table>
<thead>
<tr>
<th>Scorciatoia</th>
<th>Azione</th>
</tr>
</thead>
<tbody>
<tr>
<td>Clic</td>
<td>Seleziona un match.</td>
</tr>
<tr>
<td>Doppio clic</td>
<td>Naviga nel match.</td>
</tr>
<tr>
<td>SU, k</td>
<td>Seleziona il match precedente.</td>
</tr>
<tr>
<td>GIÙ, j</td>
<td>Seleziona il match successivo.</td>
</tr>
<tr>
<td>INVIO</td>
<td>Carica il match selezionato.</td>
</tr>
<tr>
<td>Canc</td>
<td>Elimina il match selezionato.</td>
</tr>
<tr>
<td>v</td>
<td>Vedere nel video la decisione in revisione, un secondo prima del lancio (match con una sorgente video e un segno su quella decisione).</td>
</tr>
<tr>
<td>/</td>
<td>Vai al campo filtro (giocatore, evento, torneo, data). <em>Esc</em> cancella il filtro.</td>
</tr>
<tr>
<td>Esc</td>
<td>Deseleziona/chiudi il pannello.</td>
</tr>
</tbody>
</table>
<h3>Pannello Anki (ripetizione dilazionata)</h3>
<table>
<thead>
<tr>
<th>Scorciatoia</th>
<th>Azione</th>
</tr>
</thead>
<tbody>
<tr>
<td>SPAZIO, Clic</td>
<td>Mostra la risposta (l'analisi registrata della posizione).</td>
</tr>
<tr>
<td>1</td>
<td>Valuta: Da rivedere (fallita, rivedere presto).</td>
</tr>
<tr>
<td>2</td>
<td>Valuta: Difficile.</td>
</tr>
<tr>
<td>3</td>
<td>Valuta: Bene.</td>
</tr>
<tr>
<td>4</td>
<td>Valuta: Facile.</td>
</tr>
<tr>
<td>p</td>
<td>Mostra/nascondi il pip count (identico alla scorciatoia generale, disponibile durante il ripasso).</td>
</tr>
<tr>
<td>Esc</td>
<td>Interrompi la revisione e torna all'elenco dei mazzi (ripresa possibile).</td>
</tr>
</tbody>
</table>
<h3>Pannello dei tornei</h3>
<table>
<thead>
<tr>
<th>Scorciatoia</th>
<th>Azione</th>
</tr>
</thead>
<tbody>
<tr>
<td>Doppio clic, INVIO (riga con focus)</td>
<td>Selezionare un torneo (mostrarne il dettaglio). TAB raggiunge le righe; un clic singolo evidenzia soltanto.</td>
</tr>
<tr>
<td>SU, k</td>
<td>Seleziona il torneo precedente, quando il pannello ha il focus o non è visualizzato alcun torneo diretto.</td>
</tr>
<tr>
<td>GIÙ, j</td>
<td>Seleziona il torneo successivo, quando il pannello ha il focus o non è visualizzato alcun torneo diretto.</td>
</tr>
<tr>
<td>Doppio clic (su un match del torneo)</td>
<td>Naviga nel match.</td>
</tr>
<tr>
<td>Esc</td>
<td>Annulla la modifica in corso, altrimenti cancella la ricerca di aggiunta match, altrimenti deseleziona il torneo, altrimenti chiudi il pannello (a tappe).</td>
</tr>
</tbody>
</table>
<h3>Pagina Direzione</h3>
<p>Nella pagina Direzione, <em>J</em>, <em>K</em>, <em>SU</em>, <em>GIÙ</em> e <em>INVIO</em> vanno alla coda delle proposte, tranne quando il focus è su una casella della griglia dei tavoli, dove <em>SU</em> e <em>GIÙ</em> cambiano casella. I menu contestuali sono descritti nel manuale (menu contestuali).</p>
<table>
<thead>
<tr>
<th>Scorciatoia</th>
<th>Azione</th>
</tr>
</thead>
<tbody>
<tr>
<td>Clic destro, MENU, MAIUSC-F10</td>
<td>Aprire il menu contestuale dell'oggetto con il focus: casella di tavolo, giocatore, posto del tabellone, posto, proposta, riga della cronologia. SU/GIÙ scorrono il menu, INVIO sceglie, ESC lo chiude.</td>
</tr>
<tr>
<td>SINISTRA, DESTRA, SU, GIÙ, HOME, FINE</td>
<td>Passare da una casella all'altra della griglia dei tavoli (comprese le caselle libere); la griglia occupa una sola sosta di TAB.</td>
</tr>
<tr>
<td>Da 1 a 9, poi da 0 a 9</td>
<td>Aprire la scheda del tavolo con quel numero; due cifre, entro 0,4 s, per un tavolo oltre il 9.</td>
</tr>
<tr>
<td>M, X</td>
<td>Su una casella occupata, aprire la scheda (o, se è aperta, il campo del tavolo); puntare a un tavolo occupato scambia i due incontri.</td>
</tr>
<tr>
<td>/</td>
<td>Aprire la ricerca rapida: giocatori, tavoli, match in corso e prove dell'evento, trovati per nome o per numero di tavolo (dettaglio). Nessun effetto in un campo di testo.</td>
</tr>
<tr>
<td>F11</td>
<td>Mettere la pagina Direzione a schermo intero (barra degli strumenti, schede, pannello e barra di stato nascosti), o uscirne. Anche ESC ne esce, dopo aver chiuso il menu o la scheda aperti (dettaglio).</td>
</tr>
<tr>
<td>Trascinare una casella occupata su un'altra</td>
<td>Con il mouse: su una casella libera, spostare l'incontro; su una casella occupata, scambiare i due incontri dopo conferma. ESC annulla il trascinamento.</td>
</tr>
<tr>
<td>Tutti i tavoli</td>
<td>Sulla griglia <em>Tutti i tavoli</em> di un evento, gli stessi tasti, menu e il trascinamento agiscono sul tavolo di qualsiasi prova; uno scambio con un'altra prova nomina entrambe nella conferma.</td>
</tr>
</tbody>
</table>
<h3>Pannello delle collezioni</h3>
<table>
<thead>
<tr>
<th>Scorciatoia</th>
<th>Azione</th>
</tr>
</thead>
<tbody>
<tr>
<td>Clic</td>
<td>Aggiungi/rimuovi la posizione corrente dalla collezione sotto il puntatore.</td>
</tr>
<tr>
<td>Doppio clic</td>
<td>Apri la collezione.</td>
</tr>
<tr>
<td>Canc</td>
<td>Rimuovi la posizione corrente (o le posizioni selezionate) dalla collezione aperta.</td>
</tr>
<tr>
<td>Esc</td>
<td>Torna all'elenco delle collezioni, altrimenti deseleziona la collezione, altrimenti chiudi il pannello (a tappe).</td>
</tr>
</tbody>
</table>
<h3>Pannello di trascrizione</h3>
<p>Il pannello prende questi tasti quando ha il fuoco. Davanti alla lista delle bozze ne prende già alcuni, perché riprendere il lavoro di ieri non richieda il mouse.</p>
<table>
<thead>
<tr>
<th>Scorciatoia</th>
<th>Azione</th>
</tr>
</thead>
<tbody>
<tr>
<td>GIÙ, j / SU, k (lista delle bozze)</td>
<td>Percorrere le bozze. La prima è evidenziata all'apertura: è quella modificata più di recente.</td>
</tr>
<tr>
<td>INVIO (lista delle bozze)</td>
<td>Aprire la bozza evidenziata.</td>
</tr>
<tr>
<td>n (lista delle bozze)</td>
<td>Aprire il modulo di creazione.</td>
</tr>
<tr>
<td>Clic</td>
<td>Aprire una bozza dall'elenco.</td>
</tr>
<tr>
<td>1 … 6</td>
<td>Inserire un dado. Sulla prima mossa di una partita: dado del giocatore 1, poi del giocatore 2 (il più alto comincia e gioca entrambi i dadi).</td>
</tr>
<tr>
<td>1 … 6 (tiro inserito)</td>
<td>Convalidare la mossa selezionata e aprire il lancio successivo. Su un'azione rivista, dove il cursore è posato su un'azione già scritta, la cifra ricomincia il lancio di quell'azione invece di convalidare.</td>
</tr>
<tr>
<td>1 … 6 (cursore sulla prima mossa di una partita)</td>
<td>Digitare un altro tiro d'apertura: dado del giocatore 1 poi del giocatore 2. Vince il più alto — dado grande prima, la mossa va al giocatore 1 in basso; dado piccolo prima, al giocatore 2 in alto; il primo candidato è preselezionato. Lo stesso tiro ridigitato non cambia nulla; <em>s</em> dà la mossa all'altro giocatore.</td>
</tr>
<tr>
<td>GIÙ, j</td>
<td>Selezionare il candidato successivo (le frecce della mossa appaiono sul tavoliere).</td>
</tr>
<tr>
<td>SU, k</td>
<td>Selezionare il candidato precedente.</td>
</tr>
<tr>
<td>Rotellina</td>
<td>Selezionare la candidata successiva o precedente, sia sopra la lista sia sopra la tavola: lo sguardo resta sulla tavola e le frecce scorrono.</td>
</tr>
<tr>
<td>Clic (su una riga)</td>
<td>Selezionare quel candidato.</td>
</tr>
<tr>
<td>Doppio clic (su una riga)</td>
<td>Convalidare questa candidata.</td>
</tr>
<tr>
<td>Clic (sul triangolo dei tiri)</td>
<td>Inserire il tiro con un solo gesto: la casella porta i due dadi, i doppi sulla diagonale. Sulla prima mossa di una partita il triangolo lascia il posto a una fila di sei dadi, e un clic dà il dado di un campo.</td>
</tr>
<tr>
<td>Clic, trascinamento (nessun dado inserito)</td>
<td>Giocare la mossa direttamente sul tavoliere: la pedina va dal punto cliccato alla sua destinazione, vincolata alle mosse legali, e i due dadi si deducono dai passi giocati.</td>
</tr>
<tr>
<td>Clic, trascinamento (lancio inserito)</td>
<td>Giocare la mossa sul tavoliere, vincolata alle mosse legali di quel lancio: ogni passo giocato lascia nella lista solo i candidati che lo contengono, e una mossa legale completa viene registrata subito. Su un'azione rivista, la sostituisce.</td>
</tr>
<tr>
<td>Trascinamento fuori dalle regole (lancio inserito)</td>
<td>Posare la pedina dove viene rilasciata, anche da un punto da cui non parte alcuna mossa legale, per trascrivere una mossa illegale. Il resto della mossa si gioca liberamente, con il clic come con il trascinamento, e la lista dei candidati lascia il posto a una riga che lo ricorda.</td>
</tr>
<tr>
<td>INVIO (mossa fuori dalle regole)</td>
<td>Registrare la mossa con i dadi inseriti e il tavoliere ottenuto, marcata mossa illegale se nessuna mossa legale raggiunge quel tavoliere. Una mossa fuori dalle regole non viene mai registrata da sola.</td>
</tr>
<tr>
<td>BACKSPACE (mossa in corso sul tavoliere)</td>
<td>Annullare l'ultimo passo giocato sul tavoliere. I passi rimanenti vengono rigiocati vincolati finché una mossa legale li contiene: annullare l'unico passo fuori dalle regole restituisce la lista.</td>
</tr>
<tr>
<td>INVIO</td>
<td>Registrare la mossa selezionata (ultima mossa di una partita).</td>
</tr>
<tr>
<td>BACKSPACE</td>
<td>Cancellare i due dadi immessi. È di qui che passa la ripresa di un lancio letto male, dato che una cifra convalida.</td>
</tr>
<tr>
<td>Clic (sulle caselle del lancio)</td>
<td>Cancellare i due dadi immessi, come BACKSPACE.</td>
</tr>
<tr>
<td>Esc</td>
<td>Abbandonare l'inserimento in corso.</td>
</tr>
<tr>
<td>d</td>
<td>Raddoppiare o ri-raddoppiare: la mossa selezionata viene convalidata al passaggio, con un solo tasto.</td>
</tr>
<tr>
<td>t</td>
<td>Accettare il raddoppio proposto: il cubo passa a chi accetta al valore raddoppiato e chi ha raddoppiato torna a tirare. Con il cursore su una cella, scrive l'accettazione al posto dell'azione mirata; un rifiuto diventato accettazione riapre la sua partita, e il seguito vi si digita al suo posto.</td>
</tr>
<tr>
<td>p</td>
<td>Rifiutare il raddoppio proposto: la partita è vinta al valore che il cubo aveva prima del raddoppio. Con il cursore su una cella, scrive il rifiuto al posto dell'azione mirata.</td>
</tr>
<tr>
<td>r poi 1, 2 o 3</td>
<td>Abbandonare la partita per il lato di turno: semplice, gammon o backgammon. Esc fra i due tasti annulla senza registrare nulla.</td>
</tr>
<tr>
<td>Clic (sul cubo)</td>
<td>Proporre un raddoppio per il campo di turno, come il tasto d. Davanti a un'offerta il cubo non risponde: accettare e passare sono nella riga di pulsanti.</td>
</tr>
<tr>
<td>SINISTRA, h</td>
<td>Arretrare il cursore di un'azione nella trascrizione.</td>
</tr>
<tr>
<td>DESTRA, l</td>
<td>Avanzare il cursore di un'azione.</td>
</tr>
<tr>
<td>Clic (su una cella)</td>
<td>Portare il cursore su questa azione, o sul turno mancante di un doppio turno.</td>
</tr>
<tr>
<td>Doppio clic (su una cella)</td>
<td>Digitare la mossa di quell'azione da tastiera, nella cella: 13/7 8/7*, bar/22, 6/off. Si digita solo la mossa, i dadi sono quelli della cella; INVIO la registra, anche illegale, ed Esc richiude la cella senza scrivere nulla. Vale per una mossa, una danza, una mossa non registrata, e per la cella tratteggiata dell'inserimento in corso non appena i suoi due dadi sono inseriti.</td>
</tr>
<tr>
<td>Doppio clic (sul punteggio di una partita)</td>
<td>Digitare il punteggio al quale questa partita è stata giocata, nella sua intestazione: 3-2, 3–2 o 3 2. INVIO lo registra, un campo svuotato torna al punteggio che danno le partite precedenti, ed Esc richiude il campo senza scrivere nulla. Un punteggio diverso da quello è segnalato come incoerenza. Nessun punteggio nel gioco a soldi.</td>
</tr>
<tr>
<td>Clic destro (su una cella)</td>
<td>Aprire le correzioni di questa azione: inserisci prima, inserisci dopo, elimina, cambia campo. Il cursore viene portato sulla cella al passaggio.</td>
</tr>
<tr>
<td>CTRL-INVIO</td>
<td>Terminare la bozza: scrivere la partita, o sostituire quella da cui è stata aperta, e liberare la bozza.</td>
</tr>
<tr>
<td>i</td>
<td>Inserire un'azione prima di quella del cursore (il campo proposto è quello che mantiene coerente il seguito).</td>
</tr>
<tr>
<td>a</td>
<td>Inserire un'azione dopo quella del cursore.</td>
</tr>
<tr>
<td>x, Canc</td>
<td>Eliminare la decisione in corso di modifica — l'azione del cursore, o l'immissione non ancora scritta — e arretrare sulla precedente, pronta per essere corretta; le successive mantengono il loro campo. In fondo al documento, arretra sull'ultima azione.</td>
</tr>
<tr>
<td>s</td>
<td>Dare l'azione del cursore all'altro campo.</td>
</tr>
<tr>
<td>CTRL-Z</td>
<td>Annullare l'ultimo gesto sulla bozza.</td>
</tr>
<tr>
<td>CTRL-MAIUSC-Z</td>
<td>Ripristinare il gesto annullato.</td>
</tr>
<tr>
<td>SPAZIO (video allegato)</td>
<td>Avviare o mettere in pausa il video. Senza video, SPAZIO apre la riga di comando, come altrove.</td>
</tr>
<tr>
<td>MAIUSC-SINISTRA / MAIUSC-DESTRA (video allegato)</td>
<td>Riavvolgere o avanzare il video di 5 secondi.</td>
</tr>
<tr>
<td>CTRL-MAIUSC-SINISTRA / CTRL-MAIUSC-DESTRA (video allegato)</td>
<td>Riavvolgere o avanzare il video di un secondo.</td>
</tr>
<tr>
<td>v (video allegato)</td>
<td>Fissare l'istante corrente del video come istante della mossa del cursore.</td>
</tr>
<tr>
<td>MAIUSC-V (video allegato)</td>
<td>Fissare l'istante corrente del video come istante del lancio dell'azione del cursore.</td>
</tr>
</tbody>
</table>
<p>Un tiro che non consente alcuna mossa registra la danza da sé, senza un tasto in più.</p>
<p>Con un video allegato, solo una convalida esplicita (<em>INVIO</em>, il doppio clic su un candidato, la mossa completata sulla scacchiera) fissa l'istante della mossa. La cifra del lancio successivo convalida senza istante; <em>v</em> lo fissa a posteriori. Un gesto di cubo convalida allo stesso modo, senza istante. Le frecce si leggono dalla loro posizione sulla tastiera, qualunque sia il layout. Senza video, questi tasti mantengono il loro significato abituale.</p>
<p>La cifra ha un solo senso: <strong>comincia un lancio là dove si trova il cursore</strong>. In fondo al documento non c'è nulla sotto il cursore, quindi convalida la mossa selezionata prima di aprire il lancio successivo — la mossa migliore giocata costa così i due dadi e nulla più, poiché la sua convalida è portata dal primo tasto del turno seguente. Su un'azione già scritta, alla quale si è tornati per correggerla, c'è qualcosa sotto il cursore: la cifra ricomincia il lancio di quell'azione, sul posto. La differenza si vede sullo schermo, perché la cella mirata è incorniciata nella trascrizione. L'<strong>ultima</strong> azione del documento fa eccezione: una volta ridigitato il suo lancio, la cifra successiva la convalida e apre la decisione seguente, come in fondo al documento, e anche INVIO porta lì.</p>
<p>Ciò che si sta digitando è disegnato nella trascrizione, tratteggiato, nel punto in cui sarà scritto: una correzione copre la cella che sostituisce, un inserimento apre una cella tra le sue due vicine, e il campo si legge dalla colonna. Nulla è registrato prima della convalida.</p>
<p>Un inserimento in mezzo al documento continua a inserire: la convalida apre una cella vuota di seguito, e l'azione successiva si inserisce a sua volta invece di sovrascrivere quella dopo. La fine della partita o lo spostamento del cursore vi mette fine.</p>
<p>Un gesto che non ha nulla da fare lo dice, una volta, nella barra di stato: «niente da annullare» con la pila vuota, «nessuna azione sotto il cursore» in fondo al documento. La frase si cancella da sola e rende il posto all'azione attesa.</p>
<p>Riportare il cursore su un'azione e ridigitare la corregge <strong>sul posto</strong>: la convalida sostituisce l'azione e il cursore torna dov'era — oppure, sull'ultima azione, passa in fondo al documento, dove la trascrizione continua. Se i dadi sono corretti e la mossa registrata resta una mossa legale del nuovo lancio, viene conservata; altrimenti viene proposto il primo candidato del nuovo lancio e la mossa è segnalata «da rivedere» fino alla convalida. Avanzare o arretrare il cursore dopo aver cambiato qualcosa registra la correzione al passaggio.</p>
<p>Nulla viene rifiutato né eliminato: inserire un'azione dello stesso campo della vicina crea un doppio turno, eliminare un'azione può crearne un altro, cambiare un campo può rendere illegali le mosse che seguono. Queste incoerenze sono segnalate nel transcript, mai corrette d'ufficio, e il cursore si posiziona sulla prima di esse dopo ogni gesto che scrive — tranne dove si continua: dopo un'eliminazione, un inserimento o una partita riaperta, resta dove si digita. Spostarsi o digitare un dado non lo sposta mai. Il turno che un doppio turno ha perso è una cella del transcript: h e l vi si fermano, e un lancio digitato lì viene inserito per il campo a cui mancava. La pila di annullamento vive in memoria: va persa alla chiusura della bozza.</p>
<p>Il pannello stesso — l'elenco delle bozze, la creazione, l'inserimento, la trascrizione e la barra della bozza — è descritto in Pannello Trascrizione.</p>
<h3>Provino</h3>
<table>
<thead>
<tr>
<th>Scorciatoia</th>
<th>Azione</th>
</tr>
</thead>
<tbody>
<tr>
<td>SINISTRA, DESTRA, SU, GIÙ</td>
<td>Spostarsi nella griglia.</td>
</tr>
<tr>
<td>j, k</td>
<td>Miniatura successiva, miniatura precedente.</td>
</tr>
<tr>
<td>Home, Fine</td>
<td>Prima, ultima miniatura della pagina.</td>
</tr>
<tr>
<td>Pag su, Pag giù</td>
<td>Pagina precedente, pagina successiva.</td>
</tr>
<tr>
<td>INVIO, Clic</td>
<td>Aprire la posizione sul tavoliere e chiudere il provino.</td>
</tr>
<tr>
<td>Esc</td>
<td>Chiudere il provino.</td>
</tr>
</tbody>
</table>
<h3>Pannello di aiuto</h3>
<table>
<thead>
<tr>
<th>Scorciatoia</th>
<th>Azione</th>
</tr>
</thead>
<tbody>
<tr>
<td>SINISTRA, h</td>
<td>Scheda precedente.</td>
</tr>
<tr>
<td>DESTRA, l</td>
<td>Scheda successiva.</td>
</tr>
<tr>
<td>SU, k</td>
<td>Scorri verso l'alto.</td>
</tr>
<tr>
<td>GIÙ, j</td>
<td>Scorri verso il basso.</td>
</tr>
<tr>
<td>SPAZIO</td>
<td>Pagina successiva.</td>
</tr>
<tr>
<td>Pag su</td>
<td>Inizio del contenuto.</td>
</tr>
<tr>
<td>Pag giù</td>
<td>Fine del contenuto.</td>
</tr>
<tr>
<td>/</td>
<td>Cerca nella guida: Invio passa all'occorrenza successiva, MAIUSC-Invio alla precedente.</td>
</tr>
<tr>
<td>?, CTRL-F, Esc</td>
<td>Chiudi la guida.</td>
</tr>
</tbody>
</table>
`,
    commands: `
<p>La riga di comando, situata nella barra di stato, si apre premendo il tasto <em>SPAZIO</em>. Durante la digitazione di un comando, appare automaticamente un elenco di suggerimenti: il tasto <em>TAB</em> (o <em>MAIUSC-TAB</em>) scorre le proposte e completa il comando, mentre <em>ESC</em> chiude l'elenco (un secondo <em>ESC</em> chiude la riga di comando). I tasti <em>SU</em> e <em>GIÙ</em> restano riservati alla cronologia dei comandi.</p>
<h3>Operazioni globali</h3>
<table>
<thead>
<tr>
<th>Comando</th>
<th>Azione</th>
</tr>
</thead>
<tbody>
<tr>
<td>new, ne, n</td>
<td>Crea un nuovo database.</td>
</tr>
<tr>
<td>open, op, o</td>
<td>Apre un database esistente.</td>
</tr>
<tr>
<td>import_db, idb</td>
<td>Importa e unisce un altro database.</td>
</tr>
<tr>
<td>export_db, edb</td>
<td>Esporta la selezione corrente in un nuovo database.</td>
</tr>
<tr>
<td>quit, q</td>
<td>Chiude blunderDB.</td>
</tr>
<tr>
<td>help, he, h</td>
<td>Apre la guida di blunderDB.</td>
</tr>
<tr>
<td>tutorial, tour</td>
<td>Apre il catalogo delle visite guidate dell'interfaccia.</td>
</tr>
<tr>
<td>demo</td>
<td>Carica un database di esempio (partite, torneo, raccolte, commenti, mazzo Anki, analisi) per scoprire lo strumento.</td>
</tr>
<tr>
<td>meta</td>
<td>Mostra i metadati del database.</td>
</tr>
<tr>
<td>eval, epc</td>
<td>Apre il pannello Eval (Effective Pip Count, probabilità di vittoria e verdetto di cubo in bearoff). <code>epc</code> è il vecchio nome di questo pannello, conservato.</td>
</tr>
<tr>
<td>transcribe, tr</td>
<td>Apre il pannello Trascrizione: le bozze di partite in corso di inserimento, e il modo di iniziarne una.</td>
</tr>
<tr>
<td>direct</td>
<td>Apre il pannello Tornei per dirigere un torneo, e lo richiude. Finché vi è aperta una direzione, l'area principale mostra il torneo invece del tavoliere; qualsiasi altra scheda riporta il tavoliere.</td>
</tr>
<tr>
<td>met</td>
<td>Apre la tabella di match equity Kazaross-XG2.</td>
</tr>
<tr>
<td>cm</td>
<td>Apre la matrice del cubo: il verdetto della posizione corrente a tutti i punteggi di un incontro da 5, 7 o 9 punti.</td>
</tr>
<tr>
<td>tags</td>
<td>Apre il vocabolario di tag: i tag usati in questo database, con il numero di posizioni, cliccabili per lanciare la ricerca.</td>
</tr>
<tr>
<td>log</td>
<td>Apre il registro attività: le ultime duecento righe del file di log, con il necessario per copiarle in un rapporto o aprire la cartella che le contiene.</td>
</tr>
<tr>
<td>train</td>
<td>Apre il pannello Allenamento. Con un argomento, apre e avvia: <code>train scores</code> (la scheda di punteggio di un punteggio estratto a sorte; <code>train tp</code> e <code>train takepoint</code> sono sinonimi), <code>train pips</code> (il conteggio delle pedine dei due lati), <code>train bearoff</code> (l'EPC dei due lati su una posizione generata; <code>train epc</code> è un sinonimo), <code>train evaluation</code> (la probabilità di vittoria e l'azione di cubo di una posizione generata, in partita a soldi), <code>train decision</code> (una decisione analizzata della lista sfogliata: la mossa si gioca sulla tavola, l'azione di cubo si sceglie nel pannello; <code>train quiz</code> è un sinonimo).</td>
</tr>
<tr>
<td>duel</td>
<td>Apre il pannello Duello: un match contro il Bot, con blunderDB come Arbitro (vedi Pannello Duello).</td>
</tr>
<tr>
<td>tp2</td>
<td>Apre la tabella dei takepoint con cubo a 2.</td>
</tr>
<tr>
<td>tp2_live</td>
<td>Apre la tabella dei takepoint con cubo a 2 per le corse lunghe.</td>
</tr>
<tr>
<td>tp2_last</td>
<td>Apre la tabella dei takepoint con cubo a 2 morto.</td>
</tr>
<tr>
<td>tp4</td>
<td>Apre la tabella dei takepoint con cubo a 4.</td>
</tr>
<tr>
<td>tp4_live</td>
<td>Apre la tabella dei takepoint con cubo a 4 per le corse lunghe.</td>
</tr>
<tr>
<td>tp4_last</td>
<td>Apre la tabella dei takepoint con cubo a 4 morto.</td>
</tr>
<tr>
<td>gv1</td>
<td>Apre la tabella dei valori di gammon con cubo a 1.</td>
</tr>
<tr>
<td>gv2</td>
<td>Apre la tabella dei valori di gammon con cubo a 2.</td>
</tr>
<tr>
<td>gv4</td>
<td>Apre la tabella dei valori di gammon con cubo a 4.</td>
</tr>
</tbody>
</table>
<h3>Posizioni e navigazione</h3>
<table>
<thead>
<tr>
<th>Comando</th>
<th>Azione</th>
</tr>
</thead>
<tbody>
<tr>
<td>import, i</td>
<td>Importa una o più posizioni/incontri da file (xg, xgp, sgf, mat, txt, bgf). Con un argomento — <code>import XGID=…</code> o <code>import OGID=…</code> — legge l'identificatore invece di aprire un selettore di file, per quando arriva da un messaggio, un forum o uno script.</td>
</tr>
<tr>
<td>delete, del, d</td>
<td>Elimina la posizione corrente (con conferma); la cancellazione passa dal cestino e resta annullabile per trenta giorni.</td>
</tr>
<tr>
<td>trash</td>
<td>Apre il cestino: ciò che è stato eliminato, con quanto serve a ripristinarlo.</td>
</tr>
<tr>
<td>resume</td>
<td>Elenca le importazioni che nulla ha terminato (applicazione fermata) e riprende quella scelta, indicando di nuovo la cartella o i file.</td>
</tr>
<tr>
<td>[number]</td>
<td>Vai alla posizione con l'indice indicato.</td>
</tr>
<tr>
<td>[number]%</td>
<td>Va a quella percentuale dell'elenco: <code>0%</code> la prima posizione, <code>50%</code> la metà, <code>100%</code> l'ultima.</td>
</tr>
<tr>
<td>grid, gr</td>
<td>Apre il provino: l'elenco sfogliato come griglia di mini-tavolieri, una pagina di ventiquattro alla volta; scegliere una miniatura ne apre la posizione.</td>
</tr>
<tr>
<td>list, l</td>
<td>Mostra l'analisi della posizione corrente.</td>
</tr>
<tr>
<td>comment, co</td>
<td>Mostra/scrivi commenti.</td>
</tr>
<tr>
<td>rollout, ro [fast|standard]</td>
<td>Rolla la posizione corrente (con l'impostazione scelta nella scheda <strong>gammonNet</strong> della configurazione, o il preset indicato) e apre il pannello Analisi. <code>ro search [fast|standard]</code> lo avvia sulla lista mostrata, dopo una conferma con il totale; <code>ro stop</code> ferma il rollout in corso.</td>
</tr>
<tr>
<td>history, hi</td>
<td>Apre il pannello di ricerca (la cronologia di ricerca si trova nella sua scheda <em>Cronologia</em>).</td>
</tr>
<tr>
<td>stats, st</td>
<td>Mostra/nascondi il pannello delle statistiche.</td>
</tr>
<tr>
<td>match, ma</td>
<td>Mostra/nascondi il pannello delle partite.</td>
</tr>
<tr>
<td>collection, coll</td>
<td>Mostra/nascondi il pannello delle collezioni.</td>
</tr>
<tr>
<td>lesson, le [N | edit [N]]</td>
<td>Senza argomento, elenca le lezioni del database nella barra di stato; <code>le N</code> apre la lezione N al suo primo passo; <code>le edit</code> apre l'editor delle lezioni, <code>le edit N</code> sulla lezione N (vedi Lezioni).</td>
</tr>
<tr>
<td>study, sq</td>
<td>Apre la coda di studio trasversale: i suoi blunder che nulla ha ancora trattato, dal più costoso al meno costoso (vedi I blunder che nulla ha ancora trattato).</td>
</tr>
<tr>
<td>#tag1 tag2 ...</td>
<td>Etichetta la posizione corrente.</td>
</tr>
<tr>
<td>e</td>
<td>Carica tutte le posizioni del database.</td>
</tr>
<tr>
<td>blunders, bl [n]</td>
<td>Carica gli errori peggiori (equity/MWC) nella vista di analisi, secondo il filtro statistico corrente. Un numero opzionale sceglie quanti caricarne (<code>bl 50</code>); 10 per impostazione predefinita.</td>
</tr>
<tr>
<td>m</td>
<td>Naviga nell'ultima partita visitata.</td>
</tr>
</tbody>
</table>
<h3>Modifica e ricerca</h3>
<table>
<thead>
<tr>
<th>Comando</th>
<th>Azione</th>
</tr>
</thead>
<tbody>
<tr>
<td>write, wr, w</td>
<td>Salva la posizione corrente.</td>
</tr>
<tr>
<td>write!, wr!, w!</td>
<td>Aggiorna la posizione corrente.</td>
</tr>
<tr>
<td>s</td>
<td>Cerca posizioni con i filtri.</td>
</tr>
<tr>
<td>ss</td>
<td>Cercare tra le posizioni visualizzate: risultati correnti, collezione aperta o match in revisione.</td>
</tr>
</tbody>
</table>
<h3>Filtri di ricerca</h3>
<p>Questa tabella è il riferimento della grammatica di ricerca: la riga di comando, la biblioteca di filtri e l'opzione <code>--query</code> di <code>blunderdb search</code> leggono tutte gli stessi token. La colonna <em>Equivalente CLI</em> dà, quando esiste, l'opzione di <code>search</code> che fa la stessa cosa (vedere Interfaccia a riga di comando (CLI)); un trattino segnala un filtro che solo la grammatica esprime.</p>
<p>Cinque token non portano il proprio valore: lo leggono sul tavoliere di ricerca. <code>cube</code> e <code>score</code> riprendono il cubo e il punteggio lì impostati, <code>d</code> il tipo di decisione, <code>D</code> e <code>D1</code> i dadi, <code>x</code> la struttura disegnata nella scheda <em>Tranne</em>. Un lancio non si scrive quindi mai nel token: <code>D65</code> non esiste, solo la forma di esclusione porta le sue cifre (<code>xD65</code>). Sulla riga di comando, dove non c'è tavoliere, questi token si confrontano con un tavoliere vuoto; sono le opzioni della terza colonna che occorre impiegare al loro posto.</p>
<p>Un solo token <strong>ordina</strong> invece di restringere: <code>like</code> dispone il risultato per distanza crescente da una posizione bersaglio, e tutti gli altri token restringono l'insieme così ordinato.</p>
<p>Gli errori e le equity si contano in <strong>millesimi di equity</strong> — i <em>millipoints</em> della tabella qui sotto: <code>E&gt;100</code> mantiene le mosse che sono costate almeno un decimo di punto, un punto valendo 1000 millesimi.</p>
<p>Due ricerche complete:</p>
<ul>
<li><code>s p&gt;30 w40,60 xco</code> — più di 30 pip di ritardo, tra il 40 % e il 60 % di probabilità di vittoria, nessun commento.</li>
<li><code>s ph:race E&gt;50 co:xg</code> — in corsa, una mossa che è costata almeno 50 millesimi, e un commento proveniente da eXtreme Gammon.</li>
</ul>
<table>
<thead>
<tr>
<th>Query</th>
<th>Azione</th>
<th>Equivalente CLI</th>
</tr>
</thead>
<tbody>
<tr>
<td>cube, cub, cu, c</td>
<td>La posizione verifica la configurazione del cubo.</td>
<td><code>--cube</code></td>
</tr>
<tr>
<td>score, sco, sc, s</td>
<td>La posizione verifica il punteggio.</td>
<td><code>--score1</code> <code>--score2</code></td>
</tr>
<tr>
<td>d</td>
<td>La posizione verifica il tipo di decisione (pedina o cubo).</td>
<td><code>--decision</code></td>
</tr>
<tr>
<td>dd</td>
<td>La decisione è un'azione del cubo di tipo Raddoppio / Nessun raddoppio (e non una risposta Take / Pass). Implica una decisione del cubo.</td>
<td><code>--cube-response double</code></td>
</tr>
<tr>
<td>dr</td>
<td>La decisione è una risposta Take / Pass. Implica una decisione del cubo; con <code>dd</code>, <code>dr</code> prevale.</td>
<td><code>--cube-response takepass</code></td>
</tr>
<tr>
<td>D</td>
<td>La posizione verifica il lancio dei dadi (entrambi i dadi, in qualsiasi ordine).</td>
<td><code>--dice 6,5</code></td>
</tr>
<tr>
<td>D1</td>
<td>La posizione verifica il lancio dei dadi solo sul primo dado (il valore del primo dado compare su uno dei due dadi della posizione).</td>
<td><code>--dice 6</code></td>
</tr>
<tr>
<td>xD65</td>
<td>La posizione <strong>non</strong> è stata giocata con il lancio 6-5 (in qualsiasi ordine). Il valore è indicato nel gettone; ripetibile per escludere più lanci (<code>xD65 xD54</code>).</td>
<td>—</td>
</tr>
<tr>
<td>nc</td>
<td>La posizione è senza contatto.</td>
<td>—</td>
</tr>
<tr>
<td>ph:race</td>
<td>La posizione si trova in una data fase di gioco: <code>opening</code> (apertura), <code>middlegame</code> (mediogioco), <code>race</code> (corsa) o <code>bearoff</code> (uscita delle pedine). Ripetibile (<code>ph:race ph:bearoff</code>). L'etichetta è derivata dalla tavola e non è mai modificabile; <code>blunderdb repair</code> la ricalcola.</td>
<td><code>--phase</code></td>
</tr>
<tr>
<td>gt:holding</td>
<td>La posizione rientra in un dato piano di gioco, dal punto di vista del giocatore di turno: <code>race</code>, <code>bearin</code> (rientro sotto contatto), <code>crunch</code>, <code>backgame</code>, <code>acepoint</code>, <code>blitz</code>, <code>primevprime</code>, <code>mutualholding</code>, <code>holding</code>, <code>contact</code>. Ripetibile (<code>gt:holding gt:mutualholding</code>). Etichetta derivata come la fase: calcolata dalla tavola, mai modificabile, ricalcolata da <code>blunderdb repair</code>.</td>
<td><code>--game-type</code></td>
</tr>
<tr>
<td>#prime</td>
<td>La posizione porta questo <strong>tag</strong> in uno dei suoi commenti. Un tag è una <code>#parola</code> scritta nella prosa; nulla lo dichiara. Il confronto è delimitato, quindi <code>#prime</code> non trova <code>#priming</code> — è tutta la differenza rispetto al filtro di testo, che cerca una sottostringa. Ripetibile, e i tag si <strong>sommano</strong> (<code>#prime #backgame</code> chiede entrambi): una posizione porta più tag, quindi nominarne due vuol dire «entrambi».</td>
<td>—</td>
</tr>
<tr>
<td>n&gt;x</td>
<td>La posizione è stata incontrata almeno x volte nella base — il numero di decisioni prese su di essa, tutte le partite e tutti i giocatori insieme; con <code>pl</code>, <code>pl!</code> o <code>op</code>, contano solo le occorrenze di quel giocatore. Forme <code>n&gt;3</code>, <code>n&lt;2</code>, <code>n3,10</code> e <code>n4</code> (esattamente quattro).</td>
<td>—</td>
</tr>
<tr>
<td>M</td>
<td>La posizione o quella speculare verifica i filtri.</td>
<td>—</td>
</tr>
<tr>
<td>i</td>
<td>La posizione è stata importata singolarmente, non portata da un import di partita.</td>
<td><code>--individual</code></td>
</tr>
<tr>
<td>fl</td>
<td>La posizione è stata contrassegnata nel software di origine, durante l'importazione di una partita eXtreme Gammon.</td>
<td><code>--flagged</code></td>
</tr>
<tr>
<td>x</td>
<td>La posizione non contiene alcuna pedina della struttura di esclusione (scheda <em>Tranne</em> del pannello di ricerca).</td>
<td>—</td>
</tr>
<tr>
<td>p&gt;x</td>
<td>Il giocatore ha almeno x pip di svantaggio nella corsa.</td>
<td><code>--pip-min</code></td>
</tr>
<tr>
<td>p&lt;x</td>
<td>Il giocatore ha al massimo x pip di svantaggio nella corsa.</td>
<td><code>--pip-max</code></td>
</tr>
<tr>
<td>px,y</td>
<td>Il giocatore ha tra x e y pip di svantaggio nella corsa.</td>
<td><code>--pip-min</code> <code>--pip-max</code></td>
</tr>
<tr>
<td>P&gt;x</td>
<td>Il giocatore ha una corsa di almeno x pip.</td>
<td>—</td>
</tr>
<tr>
<td>P&lt;x</td>
<td>Il giocatore ha una corsa di al massimo x pip.</td>
<td>—</td>
</tr>
<tr>
<td>Px,y</td>
<td>Il giocatore ha una corsa tra x e y pip.</td>
<td>—</td>
</tr>
<tr>
<td>e&gt;x</td>
<td>L'equity (in millipunti) della posizione è maggiore di x.</td>
<td>—</td>
</tr>
<tr>
<td>e&lt;x</td>
<td>L'equity (in millipunti) della posizione è minore di x.</td>
<td>—</td>
</tr>
<tr>
<td>ex,y</td>
<td>L'equity (in millipunti) della posizione è compresa tra x e y.</td>
<td>—</td>
</tr>
<tr>
<td>E&gt;x</td>
<td>L'errore della mossa giocata dal giocatore 1 (in millipunti) è maggiore di x.</td>
<td><code>--move-error-min</code></td>
</tr>
<tr>
<td>E&lt;x</td>
<td>L'errore della mossa giocata dal giocatore 1 (in millipunti) è minore di x.</td>
<td><code>--move-error-max</code></td>
</tr>
<tr>
<td>Ex,y</td>
<td>L'errore della mossa giocata dal giocatore 1 (in millipunti) è compreso tra x e y.</td>
<td><code>--move-error-min</code> <code>--move-error-max</code></td>
</tr>
<tr>
<td>tm&gt;x</td>
<td>Il giocatore 1 ha impiegato più di x secondi per decidere (mossa o cubo). Una mossa di durata sconosciuta non corrisponde mai. Con <code>E</code>, durata ed errore sono quelli della stessa mossa giocata.</td>
<td>—</td>
</tr>
<tr>
<td>tm&lt;x</td>
<td>Il giocatore 1 ha impiegato meno di x secondi per decidere.</td>
<td>—</td>
</tr>
<tr>
<td>tmx,y</td>
<td>Il giocatore 1 ha impiegato tra x e y secondi per decidere, estremi inclusi.</td>
<td>—</td>
</tr>
<tr>
<td>w&gt;x</td>
<td>Il giocatore ha probabilità di vittoria superiori a x %.</td>
<td><code>--winrate-min</code></td>
</tr>
<tr>
<td>w&lt;x</td>
<td>Il giocatore ha probabilità di vittoria inferiori a x %.</td>
<td><code>--winrate-max</code></td>
</tr>
<tr>
<td>wx,y</td>
<td>Il giocatore ha probabilità di vittoria comprese tra x % e y %.</td>
<td><code>--winrate-min</code> <code>--winrate-max</code></td>
</tr>
<tr>
<td>g&gt;x</td>
<td>Il giocatore ha probabilità di gammon superiori a x %.</td>
<td>—</td>
</tr>
<tr>
<td>g&lt;x</td>
<td>Il giocatore ha probabilità di gammon inferiori a x %.</td>
<td>—</td>
</tr>
<tr>
<td>gx,y</td>
<td>Il giocatore ha probabilità di gammon comprese tra x % e y %.</td>
<td>—</td>
</tr>
<tr>
<td>b&gt;x</td>
<td>Il giocatore ha probabilità di backgammon superiori a x %.</td>
<td>—</td>
</tr>
<tr>
<td>b&lt;x</td>
<td>Il giocatore ha probabilità di backgammon inferiori a x %.</td>
<td>—</td>
</tr>
<tr>
<td>bx,y</td>
<td>Il giocatore ha probabilità di backgammon comprese tra x % e y %.</td>
<td>—</td>
</tr>
<tr>
<td>W&gt;x</td>
<td>L'avversario ha probabilità di vittoria superiori a x %.</td>
<td>—</td>
</tr>
<tr>
<td>W&lt;x</td>
<td>L'avversario ha probabilità di vittoria inferiori a x %.</td>
<td>—</td>
</tr>
<tr>
<td>Wx,y</td>
<td>L'avversario ha probabilità di vittoria comprese tra x % e y %.</td>
<td>—</td>
</tr>
<tr>
<td>G&gt;x</td>
<td>L'avversario ha probabilità di gammon superiori a x %.</td>
<td>—</td>
</tr>
<tr>
<td>G&lt;x</td>
<td>L'avversario ha probabilità di gammon inferiori a x %.</td>
<td>—</td>
</tr>
<tr>
<td>Gx,y</td>
<td>L'avversario ha probabilità di gammon comprese tra x % e y %.</td>
<td>—</td>
</tr>
<tr>
<td>B&gt;x</td>
<td>L'avversario ha probabilità di backgammon superiori a x %.</td>
<td>—</td>
</tr>
<tr>
<td>B&lt;x</td>
<td>L'avversario ha probabilità di backgammon inferiori a x %.</td>
<td>—</td>
</tr>
<tr>
<td>Bx,y</td>
<td>L'avversario ha probabilità di backgammon comprese tra x % e y %.</td>
<td>—</td>
</tr>
<tr>
<td>o&gt;x</td>
<td>Il giocatore ha almeno x pedine fuori.</td>
<td><code>--off1-min</code></td>
</tr>
<tr>
<td>o&lt;x</td>
<td>Il giocatore ha al massimo x pedine fuori.</td>
<td>—</td>
</tr>
<tr>
<td>ox,y</td>
<td>Il giocatore ha tra x e y pedine fuori.</td>
<td>—</td>
</tr>
<tr>
<td>O&gt;x</td>
<td>L'avversario ha almeno x pedine fuori.</td>
<td><code>--off2-min</code></td>
</tr>
<tr>
<td>O&lt;x</td>
<td>L'avversario ha al massimo x pedine fuori.</td>
<td>—</td>
</tr>
<tr>
<td>Ox,y</td>
<td>L'avversario ha tra x e y pedine fuori.</td>
<td>—</td>
</tr>
<tr>
<td>k&gt;x</td>
<td>Il giocatore ha almeno x pedine arretrate.</td>
<td>—</td>
</tr>
<tr>
<td>k&lt;x</td>
<td>Il giocatore ha al massimo x pedine arretrate.</td>
<td>—</td>
</tr>
<tr>
<td>kx,y</td>
<td>Il giocatore ha tra x e y pedine arretrate.</td>
<td>—</td>
</tr>
<tr>
<td>K&gt;x</td>
<td>L'avversario ha almeno x pedine arretrate.</td>
<td>—</td>
</tr>
<tr>
<td>K&lt;x</td>
<td>L'avversario ha al massimo x pedine arretrate.</td>
<td>—</td>
</tr>
<tr>
<td>Kx,y</td>
<td>L'avversario ha tra x e y pedine arretrate.</td>
<td>—</td>
</tr>
<tr>
<td>z&gt;x</td>
<td>Il giocatore ha almeno x pedine nella zona.</td>
<td>—</td>
</tr>
<tr>
<td>z&lt;x</td>
<td>Il giocatore ha al massimo x pedine nella zona.</td>
<td>—</td>
</tr>
<tr>
<td>zx,y</td>
<td>Il giocatore ha tra x e y pedine nella zona.</td>
<td>—</td>
</tr>
<tr>
<td>Z&gt;x</td>
<td>L'avversario ha almeno x pedine nella zona.</td>
<td>—</td>
</tr>
<tr>
<td>Z&lt;x</td>
<td>L'avversario ha al massimo x pedine nella zona.</td>
<td>—</td>
</tr>
<tr>
<td>Zx,y</td>
<td>L'avversario ha tra x e y pedine nella zona.</td>
<td>—</td>
</tr>
<tr>
<td>bo&gt;x</td>
<td>Il giocatore ha almeno x blot nell'outfield.</td>
<td>—</td>
</tr>
<tr>
<td>bo&lt;x</td>
<td>Il giocatore ha al massimo x blot nell'outfield.</td>
<td>—</td>
</tr>
<tr>
<td>box,y</td>
<td>Il giocatore ha tra x e y blot nell'outfield.</td>
<td>—</td>
</tr>
<tr>
<td>BO&gt;x</td>
<td>L'avversario ha almeno x blot nell'outfield.</td>
<td>—</td>
</tr>
<tr>
<td>BO&lt;x</td>
<td>L'avversario ha al massimo x blot nell'outfield.</td>
<td>—</td>
</tr>
<tr>
<td>BOx,y</td>
<td>L'avversario ha tra x e y blot nell'outfield.</td>
<td>—</td>
</tr>
<tr>
<td>bj&gt;x</td>
<td>Il giocatore ha almeno x blot nel jan.</td>
<td>—</td>
</tr>
<tr>
<td>bj&lt;x</td>
<td>Il giocatore ha al massimo x blot nel jan.</td>
<td>—</td>
</tr>
<tr>
<td>bjx,y</td>
<td>Il giocatore ha tra x e y blot nel jan.</td>
<td>—</td>
</tr>
<tr>
<td>BJ&gt;x</td>
<td>L'avversario ha almeno x blot nel jan.</td>
<td>—</td>
</tr>
<tr>
<td>BJ&lt;x</td>
<td>L'avversario ha al massimo x blot nel jan.</td>
<td>—</td>
</tr>
<tr>
<td>BJx,y</td>
<td>L'avversario ha tra x e y blot nel jan.</td>
<td>—</td>
</tr>
<tr>
<td><code>t'parola1;parola2;...'</code></td>
<td>I commenti della posizione contengono almeno una delle parole.</td>
<td>—</td>
</tr>
<tr>
<td>co</td>
<td>La posizione ha un commento, qualunque sia il suo contenuto.</td>
<td><code>--has-comment</code></td>
</tr>
<tr>
<td>xco</td>
<td>La posizione non ha alcun commento.</td>
<td><code>--no-comment</code></td>
</tr>
<tr>
<td>co:user</td>
<td>La posizione porta un commento di una data provenienza: <code>user</code> (scritto da te), <code>xg</code>, <code>gnubg</code>, <code>bgf</code> (portato dall'importazione di una partita) o <code>unknown</code>. Ripetibile (<code>co:xg co:gnubg</code>).</td>
<td><code>--comment-origin</code></td>
</tr>
<tr>
<td><code>au'Alice'</code></td>
<td>La posizione ha un commento firmato da questo autore (nome intero, senza distinzione tra maiuscole e minuscole).</td>
<td><code>--comment-author</code></td>
</tr>
<tr>
<td><code>m'schema1,schema2,...'</code></td>
<td>Le migliori mosse di pedine contenenti almeno uno degli schemi.</td>
<td>—</td>
</tr>
<tr>
<td><code>m'ND,DT,DP,...'</code></td>
<td>Le migliori decisioni di cubo di No Double/Take, Double Take, Double Pass.</td>
<td>—</td>
</tr>
<tr>
<td>T&gt;x</td>
<td>Data di aggiunta della posizione al database, dopo x (AAAA/MM/GG). Non è la data della partita (<code>md</code>): l'aggiunta la fissa e l'unione di due database la conserva.</td>
<td>—</td>
</tr>
<tr>
<td>T&lt;x</td>
<td>Data di aggiunta della posizione al database, prima di x (AAAA/MM/GG). Non è la data della partita (<code>md</code>).</td>
<td>—</td>
</tr>
<tr>
<td>Tx,y</td>
<td>Data di aggiunta della posizione tra x e y (AAAA/MM/GG).</td>
<td>—</td>
</tr>
<tr>
<td>max</td>
<td>Cerca nella partita con identificatore x (es: ma3).</td>
<td><code>--match-ids</code></td>
</tr>
<tr>
<td>max,y</td>
<td>Cerca nelle partite con identificatori da x a y (es: ma2,5).</td>
<td><code>--match-ids</code></td>
</tr>
<tr>
<td>tnx</td>
<td>Cerca nel torneo con identificatore x (es: tn1).</td>
<td><code>--tournament-ids</code></td>
</tr>
<tr>
<td>tnx,y</td>
<td>Cerca nei tornei con identificatori da x a y (es: tn1,3).</td>
<td><code>--tournament-ids</code></td>
</tr>
<tr>
<td>tn'nome'</td>
<td>Cerca nei tornei il cui nome è <code>nome</code>: le maiuscole sono ignorate e <code>*</code> sostituisce qualsiasi sequenza di caratteri (es: <code>tn'open*'</code>).</td>
<td><code>--tournament-name</code></td>
</tr>
<tr>
<td>rd:x</td>
<td>Cerca nelle partite del turno x (es: <code>rd:3</code>, <code>rd:Finale</code>): il testo del turno viene confrontato senza distinguere maiuscole e minuscole, <code>*</code> sostituisce qualsiasi sequenza di caratteri. Ripetibile (<code>rd:1 rd:2</code>): l'uno o l'altro.</td>
<td><code>--round</code></td>
</tr>
<tr>
<td>ml:x</td>
<td>La partita ha una lunghezza di x punti. Forme <code>ml:7</code>, <code>ml:5,9</code>, <code>ml&gt;5</code> e <code>ml&lt;9</code> (estremi inclusi).</td>
<td><code>--match-lengths</code></td>
</tr>
<tr>
<td>md:x..y</td>
<td><strong>Data della partita</strong>, letta nella colonna <code>match_date</code> della posizione (la data della partita più antica che la raggiunge), e non la data di creazione dell'analisi (<code>T</code>). Ogni estremo è un anno, un mese o un giorno (<code>2024</code>, <code>2024-06</code>, <code>2024-06-15</code>) e copre tutta la sua durata, estremi inclusi: <code>md:2024-01..2024-12</code> va dal 1° gennaio al 31 dicembre 2024. Forme <code>md:2024</code> (tutto l'anno), <code>md&gt;2024-06</code> e <code>md&lt;2024-06</code>.</td>
<td><code>--match-date</code></td>
</tr>
<tr>
<td>idx</td>
<td>Cercare la posizione con identificativo x (es. id12).</td>
<td><code>--position-ids</code></td>
</tr>
<tr>
<td>idx,y</td>
<td>Cercare le posizioni con identificativi da x a y (es. id5,10).</td>
<td><code>--position-ids</code></td>
</tr>
<tr>
<td><code>pl'nome'</code></td>
<td>Cerca posizioni di una partita a cui ha partecipato il giocatore indicato, su entrambi i lati (es. <code>pl'Alice'</code>). Le maiuscole sono ignorate e <code>*</code> sostituisce qualsiasi sequenza di caratteri (<code>pl'Ali*'</code>).</td>
<td><code>--player</code></td>
</tr>
<tr>
<td><code>pl!'nome'</code></td>
<td>Solo le decisioni prese da questo giocatore: il giocatore di turno è quello che occupa il lato con quel nome nella partita (giocatore 1 o giocatore 2). Stesse regole di maiuscole e jolly di <code>pl</code>.</td>
<td><code>--player</code> <code>--seat-only</code></td>
</tr>
<tr>
<td><code>op'nome'</code></td>
<td>Solo le partite in cui questo giocatore è l'avversario di quello di <code>pl</code> (<code>pl'Alice' op'Bob'</code>: Alice contro Bob, su un lato o sull'altro; con <code>pl!</code>, solo le decisioni di Alice). Senza <code>pl</code>, <code>op</code> da solo designa un giocatore su entrambi i lati, come <code>pl</code>. Stesse regole di maiuscole e jolly.</td>
<td><code>--opponent</code></td>
</tr>
<tr>
<td>pr&gt;x</td>
<td>Il <strong>PR della partita</strong> del giocatore che ha preso la decisione è almeno x, letto nelle statistiche per partita del lato di quel giocatore (non il PR dell'avversario); una partita senza PR è esclusa. Forme <code>pr&gt;8</code>, <code>pr&lt;5</code> e <code>pr4,9</code>, estremi inclusi.</td>
<td><code>--pr</code></td>
</tr>
<tr>
<td>ad:xg</td>
<td>Motore e profondità dell'analisi registrata per la posizione. Motori: <code>xg</code>, <code>gnubg</code>, <code>bgblitz</code>, <code>hedgehog</code>, <code>gammonnet</code> (inizio del nome del motore, maiuscole ignorate). Profondità: <code>3ply</code> (esattamente 3 ply), <code>3ply+</code> (almeno 3 ply), <code>book</code> (libro delle aperture), <code>rollout</code> (rollout, inclusi XG Roller e Roller++). Ripetibile: i motori sono alternative, le profondità anche, e un motore e una profondità devono entrambi corrispondere (<code>ad:xg ad:gnubg ad:3ply+</code>).</td>
<td><code>--analysis</code></td>
</tr>
<tr>
<td>like, like42, like&lt;12, like42*</td>
<td>Ordina il risultato per distanza crescente da una posizione bersaglio, invece di restringerlo: <code>like</code> prende la posizione corrente, <code>like42</code> quella di indice 42, <code>like&lt;12</code> scarta ciò che dista più di dodici pip di pedina, <code>like42*</code> allarga la classe del bersaglio a tutti i tipi di decisione e a entrambi i regimi, soldi e incontro. Vedere Pannello Ricerca.</td>
<td>—</td>
</tr>
</tbody>
</table>
<p>I valori <code>pl</code>, <code>pl!</code>, <code>op</code>, <code>tn</code>, <code>m</code> e <code>t</code> si aprono con virgolette o un apostrofo e si chiudono con l'uno o l'altro. Un token che mantiene delle virgolette senza formare un valore completo viene ignorato, e <code>blunderdb search --query</code> lo segnala. Un tag può contenere un apostrofo (<code>#l'ouverture</code>), mai delle virgolette. Il punto e virgola è il separatore delle liste di identificatori e di tag: non compare in nessun valore, e <code>ma1;2</code> non è un token (scrivere <code>ma1 ma2</code>).</p>
<h3>Trascrivere da un terminale</h3>
<p>La riga di comando non ha un comando di trascrizione: i gesti si digitano nella scheda <em>Trascrizione</em>. Fuori dall'applicazione, passano per <code>blunderdb call</code> (Trascrivere tramite l'API), per esempio <code>blunderdb call transcriptions.apply --db base.db --if-match 3 --json '{"id":1,"gesture":{"Kind":"validate"}}'</code>. <code>--if-match</code> nomina la revisione restituita dalla chiamata precedente; ogni chiamata è una sessione a sé, senza <code>sessionId</code> né annullamento da una chiamata all'altra.</p>
<h3>Comandi vari</h3>
<table>
<thead>
<tr>
<th>Comando</th>
<th>Azione</th>
</tr>
</thead>
<tbody>
<tr>
<td>clear, cl</td>
<td>Cancella la cronologia dei comandi.</td>
</tr>
</tbody>
</table>
`,
    about: `
<h3>Versione</h3>
<p>Versione dell'applicazione: {appVersion}</p>
<p>Versione del database: {dbVersion}</p>
<p>
    <a href="https://kevung.github.io/blunderDB/it/" target="_blank" rel="noopener noreferrer">Documentazione in linea</a> ·
    <a href="https://kevung.github.io/blunderDB/it/historique.html" target="_blank" rel="noopener noreferrer">Cronologia delle versioni</a>
</p>

<h3>Autore</h3>
<p><strong>Kévin Unger &lt;blunderdb@proton.me&gt;</strong></p>
<p>Puoi anche trovarmi su Heroes con il nickname <strong>postmanpat</strong>.</p>
<p>
    Ho sviluppato blunderDB inizialmente per uso personale, per individuare schemi nei miei errori. Ma è molto piacevole ricevere riscontri, soprattutto quando si sono dedicate molte ore alla
    progettazione, alla programmazione, al debug... Quindi non esitare a scrivermi per condividere i tuoi riscontri.
</p>
<p>Ecco diversi modi per contattarmi:</p>
<ul>
    <li>Unisciti al server Discord di blunderDB: <a href="https://discord.gg/DA5PpzM9En" target="_blank" rel="noopener noreferrer">discord.gg/DA5PpzM9En</a>,</li>
    <li>Parla con me se ci incontriamo a un torneo,</li>
    <li>Inviami un'email,</li>
</ul>
<h3>Licenza</h3>
<p>
    blunderDB è distribuito sotto la licenza MIT. Questo significa che sei libero di usare, copiare, modificare, unire, pubblicare, distribuire, sublicenziare e/o vendere copie del software, a
    condizione che l'avviso di copyright originale e questo avviso di permesso siano inclusi in tutte le copie o porzioni sostanziali del software.
</p>
<h3>Ringraziamenti</h3>
<p>Dedico questo piccolo software alla mia compagna <strong>Anne-Claire</strong> e alla nostra cara figlia <strong>Perrine</strong>. Vorrei ringraziare in particolare alcuni amici:</p>
<ul>
    <li>
        <strong>Tristan Remille</strong>, per avermi avvicinato al backgammon con gioia e gentilezza; per avermi mostrato la Via nella comprensione di questo meraviglioso gioco; per aver continuato a
        sostenermi nonostante i miei modesti tentativi di giocare meglio.
    </li>
    <li><strong>Nicolas Harmand</strong>, un compagno gioioso per oltre un decennio in grandi avventure, e un fantastico compagno di gioco da quando si è appassionato al backgammon.</li>
</ul>
<h3>Crediti</h3>
<p>blunderDB incorpora codice, dati e caratteri di altre persone. L'essenziale:</p>
<ul>
    <li>
        La rete neurale <strong>strehl-prob5-512-512-256-256</strong> è opera di <strong>Alexander Strehl</strong> (<em>alexstrehl/backgammon-ai-engine</em>, MIT). La ricerca, il modello di cubo e la
        tabella di match equity che la circondano costituiscono la configurazione propria di <strong>gammonNet</strong> (<a
            href="https://github.com/kevung/gammonNet"
            target="_blank"
            rel="noopener noreferrer"
            >github.com/kevung/gammonNet</a
        >, MIT).
    </li>
    <li>La tabella di match equity Kazaross-XG2 (MET) è opera di <strong>Neil Kazaross</strong>.</li>
    <li>Le tabelle dei take point e dei valori di gammon sono tratte dal libro <em>The Theory of Backgammon</em> di <strong>Dirk Schiemann</strong>.</li>
    <li>
        I database di bearoff a un lato (6 punti, 15 pedine, per l'EPC) e a due lati (6 punti, 6 pedine, per i verdetti di cubo nelle corse) sono calcolati da blunderDB stesso, tramite un porting
        dello strumento <em>makebearoff</em> di <strong>GNU Backgammon</strong> (GNUbg); il risultato è identico byte per byte a quello di gnubg, la cui impronta SHA-256 fa da riferimento. GNUbg è
        software libero sotto licenza GPL.
    </li>
    <li>I file di match sono letti da <em>xgparser</em>, <em>gnubgparser</em>, <em>bgfparser</em> e <em>ogxmparser</em> (MIT).</li>
    <li>Lato Go: <em>modernc.org/sqlite</em> (BSD-3-Clause), <em>pgx</em>, <em>Wails</em> e <em>go-fsrs</em> (MIT).</li>
    <li>Lato interfaccia: <em>Svelte</em>, <em>two.js</em>, <em>Chart.js</em> e <em>driver.js</em> (MIT).</li>
    <li>I caratteri <em>Nunito</em> e <em>Noto Sans JP</em> (SIL Open Font License 1.1).</li>
</ul>
<p>
    L'inventario completo, con il testo delle licenze, è il file <strong>THIRD_PARTY.md</strong> distribuito con blunderDB (<a
        href="https://github.com/kevung/blunderDB/blob/main/THIRD_PARTY.md"
        target="_blank"
        rel="noopener noreferrer"
        >github.com/kevung/blunderDB</a
    >).
</p>
`
};
