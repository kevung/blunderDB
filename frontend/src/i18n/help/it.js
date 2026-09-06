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
<p>Le posizioni sono memorizzate in un database rappresentato da un file <em>.db</em>. L'applicazione desktop apre questo file direttamente, mai un indirizzo di rete: la modalità server (Modalità headless (server)) è un'altra modalità dello stesso binario, e si passa dall'una all'altra esportando o migrando il database, non puntando l'applicazione verso un URL.</p>
<h3>Interazioni principali</h3>
<p>Le principali interazioni possibili con blunderDB sono:</p>
<ul>
<li>aggiungere una nuova posizione,</li>
<li>modificare una posizione esistente,</li>
<li>copiare l'immagine del board negli appunti (PNG) tramite <strong>CTRL-X</strong>, oppure con l'analisi completa tramite <strong>CTRL-X CTRL-X</strong>,</li>
<li>eliminare una posizione esistente,</li>
<li>cercare una o più posizioni,</li>
<li>importare match da diverse fonti (XG, GNUbg, BGBlitz, Jellyfish), compresi i commenti dai file XG,</li>
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
<li>studiare le posizioni tramite ripetizione dilazionata (pannello Anki),</li>
<li>visualizzare i metadati del database (pannello Metadati).</li>
</ul>
<p>Possono comparire finestre modali per:</p>
<ul>
<li>visualizzare la guida di blunderDB,</li>
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
<p>Un clic destro sulla dama apre un menu contestuale che propone: valutare la posizione visualizzata nel pannello Eval, valutarne lo specchio, copiare l'immagine della dama con la sua analisi negli appunti (l'equivalente di <em>CTRL-X CTRL-X</em>, meno facile da scoprire), <strong>salvare l'immagine in un file</strong> in SVG o PNG, aprire una nuova vista su questa posizione, e — se la posizione viene già dalla base — aggiungerla a un mazzo Anki (ripetizione dilazionata).</p>
<p>Gli appunti sono il gesto corrente; salvare è l'altro bisogno — l'illustrazione di un articolo, di un messaggio di forum, di una lezione. L'<strong>SVG</strong> è proposto perché la dama lo è: è la forma che sopravvive a un ingrandimento, quella che si mette in un documento senza sfocarla. Il PNG ne deriva, come la copia negli appunti: un solo rendering, tre destinazioni, quindi nessuna può divergere dalle altre. Questo menu non compare nel pannello Eval né nel pannello Ricerca, dove il tasto destro serve già a posare le pedine dell'altro colore. Vedere Portare una posizione nel pannello Eval per portare una posizione nel pannello Eval.</p>
<p>La barra di stato è strutturata da sinistra a destra con le seguenti informazioni:</p>
<ul>
<li>la riga di comando, accessibile premendo il tasto <em>SPAZIO</em>,</li>
<li>un messaggio informativo relativo a un'operazione eseguita dall'utente,</li>
<li>l'indice della posizione corrente, seguito dal numero di posizioni nella biblioteca corrente (o le informazioni di mossa/partita durante la navigazione di un incontro),</li>
<li>il <strong>contatore della biblioteca</strong> — «412 posizioni · 38 blunder · 5 incontri» — dove ogni numero <strong>apre ciò che conta</strong>: le posizioni, la ricerca <code>E&gt;100</code> preparata nella riga di comando, o l'elenco degli incontri. Una cifra che non si può seguire è una decorazione. La soglia dei blunder è quella delle statistiche, cento millipunti: due soglie farebbero dire due cose alla stessa parola.</li>
</ul>
<div class="admonition note">
<p>Nel caso di posizioni derivanti da una ricerca dell'utente, il numero di posizioni indicato nella barra di stato corrisponde al numero di posizioni filtrate.</p>
</div>
<p>La scheda <strong>Anki</strong> porta un <strong>contrassegno</strong> quando ci sono carte da ripassare, in tutti i mazzi. Quella cifra è la ragione per aprire la scheda; non ha nulla da fare dietro di essa. Zero non mostra nulla: un contrassegno che dice «0» è rumore.</p>
<p>Il comando <code>log</code> apre il <strong>registro attività</strong>: le ultime duecento righe del file di log, un pulsante per copiarle — quanto serve per allegare un rapporto a una segnalazione — e un altro per aprire la cartella che le contiene. Il registro non è né filtrato né riformattato: un registro che si abbellisce è un registro che non si può più citare.</p>
<p>Nella <strong>cronologia delle ricerche</strong> del pannello Ricerca, ogni token di un comando salvato appare come un'etichetta con nome — <em>Senza contatto</em>, <em>Errore di mossa</em> — anziché come token nudo. Il comando esatto resta nel suggerimento, perché è quello che si rilancia; e un token che blunderDB non riconosce appare <strong>così com'è</strong>, non tradotto al più vicino.</p>
<h3>Schede delle viste</h3>
<p>Sotto la barra degli strumenti, una barra delle schede consente di lavorare con più <strong>viste</strong> in parallelo. Ogni vista è uno spazio di lavoro indipendente che conserva il proprio elenco di posizioni, l'indice della posizione corrente, la posizione visualizzata, l'analisi e la mossa selezionata, il pannello attivo, il commento in corso nonché il contesto di navigazione in un match. È così possibile, ad esempio, tenere aperta una ricerca in una vista mentre si scorre un match in un'altra.</p>
<ul>
<li><strong>Creare una vista</strong> : fare clic sul pulsante <em>+</em> della barra delle schede o premere <em>CTRL-T</em>. La nuova vista parte come copia della vista corrente.</li>
<li><strong>Chiudere una vista</strong> : fare clic sulla croce della scheda o premere <em>CTRL-W</em>. L'ultima vista non può essere chiusa.</li>
<li><strong>Cambiare vista</strong> : fare clic su una scheda, premere <em>CTRL-PageUp</em> / <em>CTRL-PageDown</em> (o <em>MAIUSC-J</em> / <em>MAIUSC-K</em>) per passare alla vista precedente / successiva, oppure da <em>CTRL-1</em> a <em>CTRL-9</em> per raggiungere direttamente l'n-esima vista.</li>
<li><strong>Rinominare una vista</strong> : fare doppio clic sulla scheda, inserire il nuovo nome e confermare con <em>INVIO</em>.</li>
</ul>
<p>Le viste vengono salvate con lo stato di sessione del database e ripristinate alla sua riapertura.</p>
<h3>Configurazione</h3>
<p>Il pulsante di configurazione (icona a forma di ingranaggio) nella barra degli strumenti, a sinistra del pulsante di aiuto, apre la finestra di configurazione di blunderDB. È organizzata in sei schede:</p>
<ul>
<li><strong>Interfaccia</strong> — lingua, scala di visualizzazione, posizione del pannello;</li>
<li><strong>Colori della board</strong> — i colori della board;</li>
<li><strong>Bearoff</strong> — le tabelle di bearoff usate dal pannello Eval;</li>
<li><strong>gammonNet</strong> — le impostazioni del valutatore integrato, descritte qui sotto;</li>
<li><strong>Cartella sorvegliata</strong> — l'importazione automatica degli incontri che arrivano in una cartella, descritta più sotto;</li>
<li><strong>Identità dell'emittente</strong> — la chiave che firma i tuoi contrassegni di origine, descritta nella sezione Distribuire un database: origine e password.</li>
</ul>
<p>La scheda <em>Interfaccia</em> comincia con un <strong>tema</strong>: <em>seguire il sistema</em>, <em>chiaro</em>, <em>scuro</em>, <em>contrasto elevato</em> o <em>stampabile</em>. Il tema regola i colori dell'interfaccia e <strong>propone una tavolozza per la dama</strong> — un'interfaccia scura attorno a una dama chiara non è un tema scuro, è la metà di uno, poiché la dama occupa gran parte della finestra.</p>
<p>Voi mantenete l'ultima parola, e il meccanismo lo garantisce anziché prometterlo: la scheda <em>Colori</em> continua a regolare la dama direttamente, e un colore scelto dopo il tema è il vostro. All'avvio sono applicati solo i token dell'interfaccia, mai la tavolozza della dama — quella che avete regolato è già caricata, e riscriverla a ogni lancio cancellerebbe il vostro lavoro una sessione alla volta. Vedere <code>ADR-0038 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0038-a-named-theme-carries-the-board-palette-and-the-user-still-has-the-last-word.md&gt;</code>__.</p>
<p><em>Seguire il sistema</em> è l'impostazione predefinita: obbedisce alla preferenza chiaro/scuro della scrivania, anche quando cambia a metà sessione. Uno strumento non impone il proprio chiaro o scuro a una scrivania che ha già deciso.</p>
<p>La scheda <em>Interfaccia</em> permette anche di scegliere la lingua fra inglese, francese, tedesco, italiano, spagnolo, finlandese, giapponese, greco e russo. Tutta l'interfaccia (barra degli strumenti, pannelli, messaggi, aiuto) è tradotta nella lingua selezionata. La scelta della lingua è salvata e conservata da una sessione all'altra.</p>
<p>La stessa scheda offre anche il pulsante <strong>Compatta il database</strong>, che recupera lo spazio su disco lasciato dalle eliminazioni (match, tornei, purghe): il database non si riduce mai da solo quando si cancellano dati, questa compattazione va chiesta esplicitamente. L'operazione può richiedere tempo su un database grande e necessita, temporaneamente, di circa il doppio della sua dimensione in spazio libero (blunderDB rifiuta di partire anziché rischiare una compattazione interrotta); prima di avviarla viene quindi chiesta una conferma. Il risultato — lo spazio guadagnato, in megabyte — appare poi nella barra di stato. La stessa operazione è disponibile da riga di comando con <code>blunderdb vacuum</code> (vedere Interfaccia a riga di comando (CLI)).</p>
<p>Il pulsante <strong>Apri la cartella dei registri</strong>, subito sotto, apre la cartella che contiene il registro dell'applicazione — utile per allegare dettagli a una segnalazione di problema, in particolare quando blunderDB è stato avviato da un collegamento o da un doppio clic, senza un terminale collegato che mostri alcunché.</p>
<p>La casella <strong>Verificare gli aggiornamenti all'avvio</strong>, disattivata per impostazione predefinita, interroga una volta per avvio la pagina delle versioni del repository GitHub e mostra nella barra di stato un messaggio se è disponibile una versione più recente — mai una finestra che blocchi l'uso. Questa verifica resta automaticamente disattivata su un'installazione fatta tramite un gestore di pacchetti (Flatpak, Homebrew, un pacchetto della distribuzione…): in quel caso è quel canale a gestire gli aggiornamenti, non blunderDB.</p>
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
<p>La scheda <em>Generale</em> porta infine <strong>Ripara le analisi</strong>: le colonne di analisi che ricerca e statistiche interrogano sono una proiezione delle analisi memorizzate, che restano intatte. Un difetto della proiezione si ripara dunque senza reimportare nulla. È esplicito e mai automatico — riscrivere le colonne di analisi di qualcuno per il solo fatto che apre la sua base non è cosa che uno strumento debba fare a sua insaputa. Lo stesso <code>blunderdb repair</code> è disponibile da riga di comando.</p>
<p>La scheda <strong>gammonNet</strong> regola il valutatore integrato (vedere <code>ADR-0011 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0011-gammonnet-is-ported-to-go-and-the-representation-boundary-sits-at-the-evaluator-s-edge.md&gt;</code>__). Vi si regolano due profondità di ricerca, denominate e conservate separatamente — abbassare l'una non modifica mai l'altra:</p>
<ul>
<li><strong>Profondità di visualizzazione</strong> — il comfort interattivo durante la modifica del tavoliere; mai scritta nel database.</li>
<li><strong>Profondità di analisi</strong> — ciò che il lotto di analisi dopo l'importazione scrive nell'Analisi di una posizione.</li>
</ul>
<p>Entrambe valgono per impostazione predefinita <strong>2-ply</strong>, la configurazione canonica. La scheda propone anche la <strong>potatura</strong> (predefinita <code>k=12</code>) e il <strong>numero di mosse candidate mostrate</strong> (predefinito 10), oltre a una casella <strong>analizza automaticamente dopo l'importazione</strong> che, una volta attivata, verifica dopo ogni importazione se restano posizioni <strong>senza alcuna analisi</strong> (né gammonNet, né XG, né GNUbg, né BGBlitz — la regola è « una valutazione colma soltanto una lacuna », mai una sostituzione) e, se è il caso, avvia in background un'analisi gammonNet alla profondità di analisi configurata. Un pulsante <strong>Analizza ora</strong> riavvia manualmente lo stesso recupero, utile per una libreria creata prima dell'esistenza di questa funzione.</p>
<p>Un secondo pulsante, <strong>Rianalizza le posizioni obsolete</strong>, copre il caso opposto: una posizione già analizzata da gammonNet, ma la cui analisi memorizzata è stata scritta con una versione del motore più vecchia di quella attualmente in esecuzione, o a una profondità diversa dalla profondità di analisi configurata sopra, viene lì segnalata come obsoleta e rivalutata. Una posizione che porta anche un'analisi XG, GNUbg o BGBlitz non viene mai toccata da questo pulsante, qualunque sia il suo contenuto gammonNet — la protezione di <code>ADR-0013 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0013-evaluations-fill-gaps-an-imported-analysis-is-never-overwritten.md&gt;</code>__ resta incondizionata. Il numero mostrato accanto a ciascun pulsante (posizioni senza analisi, posizioni obsolete) è puramente informativo; il lotto ricalcola il proprio elenco all'avvio.</p>
<p>Entrambi i lotti sono <strong>limitati, visibili e annullabili, mai un demone silenzioso</strong>: il loro avanzamento (<code>posizioni analizzate / totale</code>) e un pulsante di annullamento compaiono nella barra di stato per tutta la loro durata, e scompaiono una volta terminati a favore di un messaggio che riassume il risultato — quante posizioni sono state <strong>analizzate</strong>, quante sono state <strong>rifiutate</strong> (una posizione che gammonNet rifiuta di valutare, come un punteggio di partita fuori dalla portata della sua tabella, il che non è mai un guasto) e quante sono <strong>fallite</strong> (ritentate, invariate, alla prossima esecuzione). Chiudere l'applicazione durante l'uno o l'altro non perde nulla: ogni posizione analizzata viene scritta man mano, e la prossima esecuzione riprende esattamente da dove l'analisi si era fermata, senza alcun registro da tenere.</p>
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
<li><strong>L'importazione è silenziosa.</strong> Stavate studiando una posizione quando sono arrivati i vostri incontri: togliervi lo schermo sarebbe il momento peggiore. L'importazione avviene senza finestra, e la barra di stato mostra una fascia con il conteggio degli incontri importati, ignorati (duplicati) e falliti, con un pulsante che apre il resoconto completo se lo desiderate. Tutto il resto è identico a un'importazione manuale: stessi duplicati rilevati, stesso lotto di importazione, stessa analisi automatica se è attiva.</li>
</ul>
<p>L'intervallo predefinito è di dieci secondi; il minimo è due. La cartella non è percorsa ricorsivamente: una cartella sorvegliata è il posto dove uno strumento deposita i suoi incontri, non un albero da esplorare. Una condivisione di rete smontata non ferma la sorveglianza e non fa nemmeno passare il suo contenuto per nuovo al ritorno.</p>
<p>La stessa sorveglianza esiste da riga di comando, con <code>blunderdb import --type batch --dir &lt;cartella&gt; --watch</code> (vedere Interfaccia a riga di comando (CLI)): è la forma che un server, un'attività pianificata o uno script possono usare.</p>
<p>La finestra di configurazione raggruppa anche alcune impostazioni di visualizzazione dell'interfaccia. Un cursore di <strong>scala dell'interfaccia</strong> consente di ingrandire o ridurre l'insieme degli elementi, il che è utile sugli schermi ad alta densità o per migliorare la leggibilità. Un menu <strong>posizione dei pannelli</strong> determina la collocazione dei pannelli (ricerca, match, analisi) rispetto al tavoliere: <em>in basso</em>, <em>di lato</em> o <em>automatica</em> (il lato viene allora scelto sugli schermi larghi per sfruttare meglio lo spazio disponibile). Come le altre impostazioni, queste scelte vengono conservate da una sessione all'altra.</p>
<h3>Visite guidate e database di esempio</h3>
<p>Per facilitare i primi passi, blunderDB propone delle <strong>visite guidate</strong> dell'interfaccia. Il catalogo delle visite si apre dalla barra degli strumenti o con il comando <code>tour</code> (alias <code>tutorial</code>). Sono disponibili sette visite: una visita generale dell'interfaccia e visite dedicate alla ricerca di posizioni, alla revisione delle partite, alla revisione dei tornei, al pannello Eval, al ripasso Anki e alle statistiche. Ogni visita evidenzia gli elementi interessati dell'interfaccia, passo dopo passo, apre il pannello di cui parla, e può essere ripetuta in qualsiasi momento. Al primo avvio, la visita generale viene proposta automaticamente.</p>
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
<h3>Pannello Analisi</h3>
<p>Il pannello <strong>Analisi</strong> (<em>CTRL-L</em>) visualizza i dati di analisi della posizione corrente importati da eXtreme Gammon (XG), GNUbg o BGBlitz. Mostra le migliori alternative (mosse di pedine o decisioni di cubo) con i relativi valori di equity e gli errori corrispondenti. Il tasto <em>d</em> alterna tra l'analisi delle mosse di pedine e l'analisi del cubo. Durante la navigazione in un match, la mossa effettivamente giocata viene evidenziata nell'elenco delle alternative. Premere <em>CTRL-L</em> o eseguire il comando <code>list</code> per mostrare o nascondere il pannello.</p>
<p>Sotto le tabelle, una <strong>frase</strong> dice a volte quanto è costata la decisione giocata e perché: «Perdi 120 mMWC: la mossa giocata lascia tre pedine scoperte dove 13/7 8/7 ne lascia solo una.» Viene da sei regole misurabili — l'esposizione, un punto di casa fatto o mancato, le probabilità di gammon abbandonate, una sicurezza che costa più di quanto renda, e i due sensi di un errore di cubo (raddoppiare troppo tardi o troppo presto, prendere troppo largo o passare troppo stretto).</p>
<p>La regola che conta è quella del <strong>silenzio</strong>: la frase compare solo se una regola si applica con sicurezza, e su un errore oltre la soglia da cui i motori concordano che lo sia. Il resto del tempo non c'è frase — né cornice vuota, né «non lo sappiamo». Una spiegazione sbagliata costa più di nessuna spiegazione: insegna qualcosa di inesatto.</p>
<p>Quando una posizione è stata giudicata da <strong>più motori</strong>, una fascia in testa al pannello li mette fianco a fianco: una riga per motore, con la sua profondità e la sua risposta — il verdetto del cubo, o la sua propria mossa migliore. Dice anzitutto se concordano, ed è il disaccordo a giustificarla: «XG dice raddoppio, presa; gammonNet dice niente raddoppio» si legge a colpo d'occhio, là dove bisognava confrontare due tabelle in diagonale.</p>
<p>La mossa migliore di un motore è la migliore <strong>di quel motore</strong>: l'elenco delle mosse candidate è ordinato per equità, con tutti i motori mescolati, quindi il suo primo elemento non è la mossa migliore di nessuno in particolare.</p>
<p>La fascia appare solo se ci sono davvero più motori, ed esiste unicamente in questo pannello: il pannello Eval presenta <strong>una</strong> decisione, quella del motore integrato (<code>ADR-0017 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0017-the-panel-shows-position-facts-plus-the-one-decision-the-board-asks.md&gt;</code>__), e un confronto non vi avrebbe posto.</p>
<p>Le mosse sono scritte come si leggono sul tavoliere, qui come nel pannello Eval: la pedina meno avanzata si muove per prima, e <strong>una pedina che concatena più dadi si scrive una sola volta</strong> — un 64 giocato con la stessa pedina si legge <code>24/14</code>, e <code>24/14*</code> se colpisce all'arrivo. Il dettaglio della concatenazione ricompare solo quando dice qualcosa in più: un colpo <em>lungo il percorso</em> conserva il suo punto di passaggio, <code>24/18* 18/14</code>, senza il quale il colpo sul 18 sparirebbe dalla notazione.</p>
<p>L'equità di un'analisi importata segue la stessa regola del pannello Eval: la colonna indica il proprio referenziale, «Equity (money)» o «Equity (match)» a seconda del punteggio della posizione analizzata, mai un semplice «Equity» muto sulla scala. Le regole <strong>Jacoby</strong> e <strong>Beaver</strong> attive su una posizione money game vengono mostrate anch'esse, in badge sotto la tabella di decisione del cubo.</p>
<h3>Pannello Commenti</h3>
<p>Il pannello <strong>Commenti</strong> (<em>CTRL-P</em>) mostra, aggiunge e modifica i commenti associati alla posizione corrente. Una posizione può portarne più d'uno: sono mostrati tutti, dal più recente al più antico. I commenti importati dai file XG vengono associati automaticamente alle posizioni corrispondenti. Premere <em>CTRL-P</em> o eseguire il comando <code>comment</code> per mostrare o nascondere il pannello.</p>
<p>Ogni commento proveniente da un file porta un'<strong>etichetta di provenienza</strong> (<code>XG</code>, <code>GNU BG</code>, <code>BGF</code>, oppure <em>importato</em> quando la provenienza non è mai stata registrata). I commenti che hai scritto non ne portano: è il caso normale, e segnalarlo a ogni riga sarebbe rumore. Modificare un commento importato te lo attribuisce: dopo la modifica, la frase è la tua.</p>
<p>Questa distinzione si vede altrove: cancellare una partita non distrugge più una posizione su cui <strong>tu</strong> avevi scritto. Una nota ripresa dal file di origine, invece, sparisce con la partita che l'ha portata.</p>
<h4>I tag</h4>
<p>Un <strong>tag</strong> è una <code>#parola</code> scritta in un commento. Nulla lo dichiara, nessuna tabella lo porta, ed è voluto: il vocabolario è la vostra prosa, ed esigere una dichiarazione prima di poter taggare trasformerebbe un'abitudine in scartoffie.</p>
<p>Ciò che mancava era l'altra metà: <strong>vedere</strong> il vocabolario che ci si è costruito, e cliccare un tag invece di ricordare come lo si scriveva. Il comando <code>tags</code>, o il pulsante <code>#</code> accanto alla casella di scrittura, apre la finestra del vocabolario: i tag di questo database, ognuno con il <strong>numero di posizioni</strong> che lo portano, cliccabili per lanciare la ricerca corrispondente. Sotto l'elenco figurano i tag consigliati che il database non usa ancora — un vocabolario tratto dalla letteratura del backgammon (<code>#blitz</code>, <code>#prime</code>, <code>#holding</code>, <code>#backgame</code>, <code>#containment</code>, <code>#crunch</code>, <code>#ace-point</code>, <code>#timing</code>…), suggerito e mai imposto: un tag assente da quell'elenco vale esattamente quanto uno che vi figura.</p>
<p>Mentre si scrive, un <code>#</code> propone i tag che <strong>questo database</strong> usa già, poi quelli consigliati. È ciò che evita di scrivere <code>#back-game</code> un giorno e <code>#backgame</code> il giorno dopo, cosa che nient'altro coglierebbe.</p>
<p>La ricerca per tag si scrive <code>#prime</code> nella riga di comando. È <strong>delimitata</strong>: <code>#prime</code> non trova <code>#priming</code>, là dove una ricerca di testo ordinaria, che cerca una sottostringa, non sa distinguerli. Più tag si <strong>sommano</strong> — <code>s #prime #backgame</code> chiede le posizioni che portano entrambi — perché una posizione porta più tag: nominarne due non può che voler dire «entrambi». È l'opposto del filtro di fase o di provenienza, dove una posizione ha un solo valore e nominare due valori non può che voler dire «l'uno o l'altro».</p>
<p>La stessa lista si ottiene fuori dall'interfaccia con <code>blunderdb list --type tags</code> (vedere Interfaccia a riga di comando (CLI)).</p>
<h3>Il cestino</h3>
<p>Eliminare una posizione, una collezione o un commento passa ora da un <strong>cestino</strong>: l'eliminazione avviene davvero, ma una copia di ciò che sparisce è conservata trenta giorni. Il comando <code>trash</code> apre la finestra che le elenca, ciascuna con <em>Ripristina</em> ed <em>Elimina</em>.</p>
<p>Una posizione ripristinata torna con <strong>la sua analisi e i suoi commenti</strong> — restituirla nuda sarebbe un ripristino solo di nome. Non torna col suo vecchio numero: la riga d'origine non esiste più, e blunderDB la risalva tramite la sua impronta, il che garantisce che non crei mai un doppione ma le dà un nuovo identificativo. Una collezione torna con la sua lista; le posizioni che conteneva non erano mai state eliminate — una collezione è una vista su di esse.</p>
<p>Ciò che ha più di trenta giorni viene eliminato dal comando <code>vacuum</code>, mai dall'apertura di una base: non fare <code>vacuum</code> è tenere tutto.</p>
<div class="admonition note">
<p>Il cestino non viaggia. Un'esportazione non lo porta, ed eliminare una partita non ci mette nulla: la pulizia delle posizioni orfane che segue l'eliminazione di una partita è una manutenzione automatica, non un gesto dell'utente — vedi la regola di ritenzione in Pannello Match.</p>
</div>
<h3>Pannello Ricerca</h3>
<p>Il pannello <strong>Ricerca</strong> (<em>CTRL-F</em> o <em>TAB</em>) permette di filtrare le posizioni secondo criteri liberamente combinabili: struttura delle pedine, tipo di decisione di cubo, magnitudo dell'errore, date, tag, ecc. Il tasto <em>TAB</em> apre contemporaneamente il pannello di ricerca e l'editor di posizione, consentendo di definire una struttura di pedine da cercare direttamente sul board.</p>
<p>Per affinare una ricerca tra le posizioni attualmente filtrate, usare il comando <code>ss</code> seguito da filtri (es.: <code>ss nc</code>, <code>ss E&gt;40</code>). Il pannello di ricerca offre anche una casella di spunta <em>Cerca nei risultati correnti</em> per la stessa funzionalità.</p>
<p>Il pannello offre un controllo esplicito del <strong>tipo di decisione</strong> ricercato: <em>Indifferente</em> (nessun filtro), <em>Pedine</em> (decisioni di mossa) o <em>Cubo</em> (decisioni di cubo). Quando è selezionato <em>Cubo</em>, un secondo elenco precisa il sotto-tipo: <em>Tutti</em>, <em>Raddoppio / No raddoppio</em> (il giocatore di turno deve decidere se raddoppiare) o <em>Accetta / Passa</em> (risposta a un raddoppio avversario). Il controllo è sincronizzato con il board: modificare i dadi o il cubo sul board aggiorna il tipo di decisione, e viceversa. In modalità <em>Accetta / Passa</em>, il cubo è mostrato al centro del board al valore offerto; tale valore resta modificabile.</p>
<p>La <strong>fase di gioco</strong> — apertura, mediogioco, corsa, uscita delle pedine — è un'etichetta che blunderDB calcola dalla sola tavola. Non è mai modificabile ed è cercabile tramite il token <code>ph:</code> della riga di comando (<code>ph:race</code>, ripetibile: <code>ph:race ph:bearoff</code>). Tre delle sue quattro frontiere sono quelle che GNU Backgammon usa per indirizzare le sue reti; la quarta, dove finisce l'apertura, è una convenzione di blunderDB: una posizione è ancora in apertura finché nessuno dei due campi ha mosso più di quattro pedine dai propri punti di partenza, nessuna è uscita e nessuna è sulla barra.</p>
<div class="admonition note">
<p>L'etichetta viene ricalcolata dal comando <code>blunderdb repair</code>. Su una base aperta per la prima volta con questa versione, il calcolo avviene una volta, all'apertura. Una base le cui fasi non sono mai state calcolate non restituisce nulla per <code>ph:</code> — nulla, piuttosto che una risposta sbagliata.</p>
</div>
<p>Il comando <code>like</code> risponde a una domanda diversa da quella dei token: sostituisce la lista percorsa con le posizioni più <strong>vicine</strong> a quella corrente, dalla più vicina alla più lontana. La vicinanza è una distanza di trasporto, espressa in pip di pedina — la quantità di movimento di pedine che separa le due posizioni — e il punto di vista è sempre quello del giocatore di turno. Non è un filtro: la somiglianza <strong>ordina</strong> l'intera biblioteca invece di restringerla, e quindi non si combina con i token.</p>
<p>Il token <code>n</code> conta gli <strong>incontri</strong>: <code>n&gt;3</code> tiene le posizioni a cui arrivano più di tre mosse, in tutti gli incontri. È un'altra domanda rispetto a «cosa ho sbagliato» — una posizione incontrata venti volte e giocata bene diciannove resta quella da sapere a memoria. Si contano le mosse, non gli incontri: la stessa posizione due volte in un incontro conta due, perché erano due decisioni.</p>
<p>Una frase a parole può sostituire i token, con il comando <code>ask</code>: <code>ask my cube blunders at a score</code>. La frase è <strong>tradotta in token</strong>, scritti nella barra dei comandi — si rileggono, poi si lancia. Nulla è indovinato e nulla lascia la macchina: il vocabolario è fisso, la stessa frase dà sempre la stessa interrogazione, e ciò che non è stato compreso viene <strong>detto</strong> anziché passato sotto silenzio. Una traduzione sbagliata si vede così prima di restituire risultati sbagliati, e i token si imparano leggendoli.</p>
<p>Due intenzioni non sono token e si pongono sulla tavola di ricerca anziché nella riga: «cubo» o «pedine» (il tipo di decisione) e «al punteggio» o «money». <code>ask</code> le pone lì.</p>
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
<p>Il pannello di ricerca presenta tre schede sul bordo sinistro: <em>Ricerca</em> (i filtri), <em>Cronologia</em> e <em>Salvati</em>. La scheda <strong>Cronologia</strong> elenca le ricerche passate con la loro data e il loro comando: un clic seleziona una ricerca e mostra la posizione associata sul tavoliere, un doppio clic la riesegue. Ogni voce può essere salvata nella libreria di filtri (icona segnalibro, assegnando un nome al filtro) o eliminata. La scheda <strong>Salvati</strong> contiene la <strong>libreria di filtri</strong> : fare doppio clic su un filtro salvato per rilanciare la ricerca corrispondente (vedere Appendice: Utilizzo avanzato dei filtri). Il comando <code>history</code> (alias <code>hi</code>) apre il pannello di ricerca.</p>
<div class="admonition tip">
<p>Fare riferimento a elenco dei comandi per l'elenco dei filtri disponibili.</p>
</div>
<h3>Pannello Raccolte</h3>
<p>Il pannello <strong>Collezioni</strong> (<em>CTRL-B</em>) consente di gestire collezioni di posizioni. Le collezioni possono essere create, rinominate ed eliminate. Vi si possono aggiungere o togliere posizioni (tasto <em>Canc</em>, viene chiesta conferma). Fare doppio clic su una collezione per scorrerne le posizioni con i tasti <em>SINISTRA</em> e <em>DESTRA</em>. L'ordine delle collezioni e delle posizioni all'interno di una collezione può essere modificato per trascinamento. Premere <em>CTRL-B</em> o eseguire il comando <code>collection</code> per mostrare o nascondere il pannello.</p>
<h3>Importazione: cosa viene scritto, cosa non lo è mai</h3>
<p>Importare un match, una posizione o un altro database aggiunge ciò che manca; non sostituisce ciò che è già presente.</p>
<ul>
<li><strong>Una posizione non è mai duplicata.</strong> È la sua identità — pedine, cubo, dadi, punteggio — a riconoscerla, mai il file da cui proviene: la stessa posizione incontrata in due match resta un'unica riga.</li>
<li><strong>Un'analisi per motore.</strong> eXtreme Gammon, GNUbg, BGBlitz e il valutatore integrato convivono sulla stessa posizione, e il pannello Analisi indica l'origine di ciascuna. Importarne una non cancella l'altra.</li>
<li><strong>Un'analisi importata non viene mai ricalcolata.</strong> blunderDB la conserva così com'è, con la sua etichetta di livello (« 3-ply », « XG Roller++ », « Book »), le sue equità, i suoi errori, le sue probabilità e la fortuna del lancio. La regola è « una valutazione colma soltanto una lacuna »: l'analisi automatica dopo l'importazione visita solo le posizioni senza <strong>alcuna</strong> analisi, e <em>Rianalizza le posizioni obsolete</em> lascia intatta ogni posizione che porta un'analisi importata (vedere Configurazione).</li>
<li><strong>Reimportare lo stesso file non riscrive nulla.</strong> Il match viene riconosciuto come già presente; vengono aggiunti solo i contrassegni posti nel software di origine, senza toccare i commenti né le analisi.</li>
<li><strong>Ciò che blunderDB non scrive mai</strong>: una fortuna ricalcolata — viene letta nel file sorgente, oppure resta sconosciuta — e un rollout, di cui non apre i dati in un file <code>.xg</code> e che non sa produrre.</li>
</ul>
<p>Una raccolta può essere <strong>viva</strong>: il suo contenuto non è più una lista fatta a mano ma il risultato di una <strong>ricerca</strong>, rivalutato ogni volta che la si apre. Il pulsante ◇ in testa alla raccolta la rende viva con l'ultima ricerca lanciata; ◈ segnala che lo è già, e lo stesso pulsante le restituisce la lista. Nulla viene distrutto: le posizioni che conteneva sono ancora lì quando si torna indietro.</p>
<p>Una raccolta viva la cui interrogazione porta un token che questa versione non conosce più <strong>rifiuta di aprirsi</strong> e lo dice, invece di restituire l'intera base. È l'unico guasto che un filtro salvato non deve avere: allargarsi in silenzio.</p>
<h3>Pannello Match</h3>
<p>Il pannello <strong>Match</strong> (<em>CTRL-Tab</em>) elenca i match importati. Fare doppio clic su un match (o premere <em>INVIO</em>) per navigare tra le sue mosse. Il comando <code>m</code> riprende la navigazione nell'ultimo match visitato.</p>
<p>L'utente può:</p>
<ul>
<li>scorrere le mosse di un match usando i tasti <em>SINISTRA</em> e <em>DESTRA</em>,</li>
<li>passare da una partita all'altra con i tasti <em>PageUp</em> e <em>PageDown</em>,</li>
<li>visualizzare l'analisi delle mosse (pedine e cubo) premendo <em>CTRL-L</em>,</li>
<li>alternare tra l'analisi delle mosse di pedine e quella del cubo con il tasto <em>d</em>,</li>
<li>vedere la mossa effettivamente giocata evidenziata nell'analisi.</li>
</ul>
<p>L'ultima posizione visitata in ciascun match viene memorizzata e ripristinata automaticamente. Premere <em>CTRL-Tab</em> o eseguire il comando <code>match</code> per mostrare o nascondere il pannello.</p>
<p>Il pulsante <strong>⊕</strong> di una riga arricchisce quell'incontro da un file. Dietro non c'è nulla di nuovo: reimportare lo stesso incontro in un altro formato lo arricchisce già sul posto — l'impronta canonica riconosce che si tratta dello stesso incontro, e le analisi e i commenti del secondo file completano il primo. Ciò che il pulsante apporta è che lo si trova: nessuno indovina che un'importazione è anche un arricchimento. Il resoconto che segue dice quale dei due è avvenuto — «arricchiti: 1» invece di «importati: 1».</p>
<p>Ogni match può essere esportato in trascrizione Jellyfish <code>.mat</code> tramite il pulsante ⬇ dell'elenco dei match o il pulsante <em>.mat</em> della scheda del match.</p>
<p>Il pulsante <strong>Unisci giocatori</strong> della barra degli strumenti del pannello apre una finestra che elenca tutti i nomi dei giocatori del database con il loro numero di match: selezionare le varianti di ortografia di uno stesso giocatore, scegliere il nome canonico da conservare, quindi unire. Utile per unificare le statistiche per giocatore quando uno stesso giocatore compare con più nomi.</p>
<p>Quando un match è aperto, una <strong>barra delle informazioni</strong> compare sopra il tavoliere: ricorda i giocatori presenti (<em>giocatore 1</em> contro <em>giocatore 2</em>) nonché il contesto del match (evento, luogo, turno, data e lunghezza del match, quando queste informazioni sono disponibili). Questa barra viene mostrata anche al di fuori della modalità match: quando una posizione studiata (proveniente da una ricerca, da una collezione o da un accesso diretto) proviene da uno o più match, ne indica la <strong>provenienza</strong> — il primo match interessato e, se del caso, un badge « +N » che elenca gli altri al passaggio del mouse. Una posizione importata da sola, che nessun match referenzia, non mostra nulla.</p>
<p>All'apertura di un database contenente match, il pannello <strong>Match</strong> viene mostrato subito e la revisione inizia direttamente sulla prima posizione, così da cominciare immediatamente la navigazione.</p>
<div class="admonition note">
<p>Un database può essere aperto in scrittura da una sola finestra alla volta. Se si apre un database già aperto in un'altra finestra di blunderDB, esso si apre in <strong>sola lettura</strong> : la navigazione, la ricerca e l'analisi restano possibili, ma qualsiasi modifica è disattivata e la barra del titolo mostra « [sola lettura] ».</p>
</div>
<div class="admonition tip">
<p>Fare riferimento a Scorciatoie da tastiera per le scorciatoie disponibili.</p>
</div>
<h3>Pannello Tornei</h3>
<p>Il pannello <strong>Tornei</strong> (<em>CTRL-Y</em>) permette di raggruppare i match in tornei per un monitoraggio organizzato e un'analisi statistica per evento. I tornei possono essere creati, rinominati ed eliminati; i match possono essere assegnati ad essi. Le statistiche del pannello Stats possono essere filtrate per torneo. Premere <em>CTRL-Y</em> per mostrare o nascondere il pannello.</p>
<p>I tornei si riempiono da soli all'importazione. I file XG, GnuBG e BGF nominano il loro evento; quando un match nuovo viene importato, blunderDB lo classifica nel torneo con quel nome e lo crea se non esiste ancora. La data e il luogo del torneo restano vuoti: è qui che si compilano. Un match già presente nel database non viene mai riclassificato: reimportarne il file non disfa la sistemazione fatta a mano.</p>
<p>La colonna <strong>PR</strong> di ogni torneo mostra il PR del <strong>giocatore di riferimento</strong> — vale a dire il giocatore presente nel maggior numero di partite del torneo (in caso di parità, quello che ha preso più decisioni). Il PR non mescola quindi il vostro gioco con quello dei vostri avversari: per i vostri tornei, riflette la vostra sola prestazione. Il nome del giocatore di riferimento appare in un suggerimento passando sopra il valore.</p>
<h3>Pannello Stats</h3>
<h4>Introduzione</h4>
<p>Il pannello <strong>Stats</strong> permette di analizzare il proprio livello di gioco e di seguire la propria progressione nel tempo a partire dalle posizioni importate nel database. Calcola e visualizza gli indicatori <strong>PR</strong> (<em>Performance Rating</em>) e <strong>MWC cost</strong> (Match Winning Chance cost) per l'insieme delle posizioni o per un sottoinsieme filtrato.</p>
<p>Il pannello Stats è particolarmente utile per:</p>
<ul>
<li><strong>collocare il proprio livello</strong> rispetto alle fasce di livello (<em>Classe mondiale</em>, <em>Esperto</em>, *Avanzato*…) grazie al PR globale;</li>
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
</ul>
<p>Un pulsante <strong>Reset</strong> azzera tutti i filtri (tranne il giocatore rilevato automaticamente).</p>
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
<p>Probabilità cumulata di vittoria del match persa a causa degli errori, sull'intero set di dati filtrato. Calcolata a partire dalla MET Kazaross-XG2 integrata in blunderDB.</p>
<div class="admonition caution">
<p>Il MWC cost <strong>non è applicabile</strong> alle posizioni <em>money-game</em> (senza posta di match). Queste posizioni sono escluse dal calcolo MWC. I valori MWC dipendono dalla MET utilizzata; non sono direttamente confrontabili tra software che usano MET diverse.</p>
</div>
</blockquote>
<p>La commutazione PR ↔ MWC è istantanea: non viene eseguito alcun ricalcolo da parte del backend.</p>
<h4>Il rapporto HTML</h4>
<p>Il pulsante <strong>Rapporto HTML</strong> nell'intestazione del pannello produce un documento <strong>autonomo</strong>: un solo file, senza immagini esterne, senza foglio di stile remoto, senza script. I diagrammi sono SVG in linea, disegnati dallo stesso rendering della dama a schermo, con la vostra tavolozza. Si apre in qualunque browser, viaggia per posta elettronica, e <strong>si stampa in PDF dal browser stesso</strong> — il che evita di imbarcare un generatore di PDF per produrre ciò che tutti hanno già.</p>
<p>Contiene gli indicatori del perimetro corrente (posizioni, incontri, decisioni contate, PR globale, pedine e cubo), poi le <strong>dieci decisioni più costose</strong>, ciascuna con il suo diagramma, il suo costo, l'incontro da cui viene e la mossa migliore quando un'analisi la fornisce.</p>
<p>Il rapporto porta il <strong>filtro corrente</strong> del pannello Stats. Un rapporto che non dichiara il proprio perimetro è un rapporto le cui cifre non vogliono dire nulla: regolate il filtro — un torneo, un intervallo di date, un giocatore — prima di produrlo.</p>
<h4>Scheda Dashboard</h4>
<p>La scheda <strong>Dashboard</strong> offre una visione sintetica degli indicatori chiave.</p>
<h5>Schede di livello</h5>
<p>Tre schede visualizzano il PR (o MWC) per:</p>
<ul>
<li><strong>PR Globale</strong> — tutte le decisioni (pedine + cubo);</li>
<li><strong>PR Pedina</strong> — solo mosse di pedine;</li>
<li><strong>PR Cubo</strong> — solo decisioni di cubo.</li>
</ul>
<p>Cliccando su una scheda si caricano nel pannello di analisi le posizioni del sottoinsieme corrispondente (drill-down).</p>
<div class="admonition note">
<p>Il numero totale di decisioni viene visualizzato in fondo a ciascuna scheda al passaggio del mouse.</p>
</div>
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
<h4>Scheda Ripartizioni</h4>
<p>La scheda <strong>Ripartizioni</strong> divide le stesse decisioni che contano le cifre globali lungo quattro assi. Nessuno di essi ridefinisce cosa conta come decisione: sarebbe un secondo PR con lo stesso nome.</p>
<ul>
<li><strong>Per fase di gioco</strong> — apertura, mediogioco, corsa, uscita delle pedine. È ciò che risponde a «il mio PR in corsa contro il mio PR in contatto». L'etichetta è calcolata dalla tavola (vedi Pannello Ricerca); una base le cui fasi non sono mai state calcolate mette tutto sotto <em>Non classificata</em>, e <code>blunderdb repair</code> la riempie.</li>
<li><strong>Per piano di gioco</strong> — corsa, blitz, ancora, backgame, blocco contro blocco… È la ripartizione per cui il classificatore esiste: «dove perdo di più?», piano per piano. La stessa etichetta derivata della fase, le stesse riserve, e <code>blunderdb repair</code> la riempie allo stesso modo.</li>
<li><strong>Per etichetta</strong> — i <code>#parola</code> scritti nei commenti. Una posizione può portarne più d'una: <strong>queste righe non sommano al totale</strong>, e il pannello lo dice sotto la tabella. Un'etichetta qualifica, non partiziona.</li>
<li><strong>Per punteggio</strong> — i punti mancanti a entrambi i campi, letti dal lato del giocatore di turno, quindi dal lato di chi decide. La riga <em>Money</em> è la partita a soldi. Una cella con meno di dieci decisioni è <strong>in grigio con il suo effettivo visibile</strong> invece che nascosta: troppo poco per essere letta, ma l'omissione resta verificabile.</li>
</ul>
<div class="admonition note">
<p>La partita Crawford non è distinta: blunderDB non registra questo indicatore su una posizione. L'effetto pratico è modesto — una partita Crawford non ha alcuna decisione di cubo — ma l'omissione è reale e vale meglio scriverla che lasciarla indovinare.</p>
</div>
<h4>Studio e gioco reale</h4>
<p>Il comando <code>blunderdb list --type study --days 30</code> mette tre numeri uno accanto all'altro, piano per piano: quante <strong>posizioni distinte</strong> sono state ripassate nel periodo, quale era il PR <strong>prima</strong>, quale è il PR <strong>da allora</strong>.</p>
<p>Tre numeri, e nessun quarto. <strong>Non c'è colonna di guadagno né freccia</strong>, perché qui nulla controlla nulla: il giocatore può aver incontrato avversari più forti, cambiato formato, o semplicemente giocato più corse questo mese. L'accostamento è del lettore; una colonna che annunciasse un effetto affermerebbe una causalità che questi dati non portano. I numeri, invece, sono esatti.</p>
<p>I ripassi sono contati in <strong>posizioni distinte</strong>: una carta ripassata quattro volte nel mese è una posizione studiata, e contare le ripetizioni farebbe sembrare un mese di sgobbata un mese di copertura. Le decisioni del PR, invece, sono contate tutte — ciascuna è stata presa una volta. Un PR che poggia su meno di dieci decisioni mostra <code>—</code>, con il suo campione visibile accanto.</p>
<h4>Scheda Giocatori</h4>
<p>Le quattro schede precedenti descrivono <strong>un</strong> giocatore; la scheda <strong>Giocatori</strong> li confronta tutti. Mostra una riga per giocatore della base, il che risponde al bisogno di un organizzatore che segue un'intera competizione più che un singolo giocatore.</p>
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
<td>Il nome <strong>così come figura nei match</strong>. Un giocatore registrato con due grafie appare quindi su due righe; usa la fusione dei giocatori per riunirle.</td>
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
<td>Numero di errori gravi (almeno 0,100 EMG).</td>
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
<li><strong>Aprire il dettaglio di un giocatore</strong> — fare clic su una riga. Il giocatore viene selezionato nella barra dei filtri e la visualizzazione passa alla scheda Dashboard.</li>
<li><strong>Restringere il periodo</strong> — i filtri di date, tornei e lunghezza dei match si applicano normalmente, il che consente di delimitare la tabella alle date di una competizione.</li>
</ul>
<div class="admonition note">
<p>In questa scheda l'elenco <strong>Giocatore</strong> e la scelta del <strong>tipo di decisione</strong> sono disattivati: la tabella mostra tutti i giocatori e ripartisce già le decisioni di pedine e di cubo in colonne distinte.</p>
</div>
<div class="admonition important">
<p>Un trattino («—») segnala un valore <strong>mai misurato</strong>, da non confondere con zero. È in particolare il caso della colonna Fortuna per ogni match importato prima della versione 2.15.0 dello schema: la fortuna non veniva allora conservata, e nulla ne consente la ricostruzione a posteriori — occorre reimportare i file di origine. I formati che non la trasportano (BGF, Jellyfish <code>.mat</code>) non la forniranno mai.</p>
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
<li>Il MWC cost viene calcolato a partire dalla <strong>MET Kazaross-XG2</strong>, tabella di riferimento de facto nel backgammon competitivo. I risultati non sono direttamente comparabili con software che utilizzano altre MET. È la stessa tabella, letta dallo stesso punto di ingresso, di cui si serve il valutatore integrato per le sue decisioni di cubo al punteggio: le statistiche e il motore non possono divergere su questo punto. Fornisce i propri valori fino a 25 punti da fare per parte; oltre, viene prolungata da una tabella di Zadeh calcolata come quella di GNUbg, fino a 64.</li>
<li>Le posizioni <em>money-game</em> (senza punteggio di match) sono <strong>escluse</strong> dal calcolo MWC. Se il database contiene molte posizioni money-game, il MWC cost potrebbe essere sottostimato o non disponibile.</li>
<li>Il MWC cost è cumulativo sull'intero set di dati filtrato — non è un indicatore per singola decisione. Misura l'impatto totale dei tuoi errori sulle tue probabilità di vittoria.</li>
</ul>
<h3>Pannello Eval</h3>
<p>Il pannello <strong>Eval</strong> (<em>CTRL-E</em>) valuta in tempo reale qualunque posizione si trovi sul tavoliere; su una posizione di bearoff si specializza e calcola inoltre l'EPC (Effective Pip Count). Si attiva premendo <em>CTRL-E</em>, facendo clic sulla scheda Eval del pannello inferiore, oppure eseguendo il comando <code>epc</code>. Questo comando conserva il suo nome originario: il pannello si è chiamato <em>EPC</em>, poi <em>Bearoff</em>, prima di diventare <em>Eval</em> — è dunque qui che va cercato ciò che una versione precedente chiamava il pannello Bearoff, il nome non designando più che la scheda di configurazione delle tabelle di uscita.</p>
<p>Il pannello mostra sempre l'<strong>unica decisione</strong> che la posizione posta sul tavoliere richiede — mai due alla volta — e i fatti che l'accompagnano. Ogni grandezza si legge nell'asse che le conviene anziché in un asse unico imposto: la probabilità di vittoria, di gammon, di backgammon e l'equità cubeless di ciascun giocatore, calcolate <em>prima del lancio</em>, si leggono <strong>per giocatore</strong> (basso, alto, poi Δ), a sinistra della decisione di cubo, quando nessun dado è impostato. I fatti e la decisione restano fianco a fianco: la decisione di cubo non finisce mai sotto i numeri che la giustificano, qualunque siano la lingua dell'interfaccia e la posizione sul tavoliere. Non appena dei dadi sono impostati, questi stessi valori <em>prima del lancio</em> cambiano asse: si leggono <strong>al tiro</strong>, in testa alla lista delle mosse candidate, sotto forma di una riga in corsivo <em>prima del lancio</em> — non una mossa candidata in più, ma un riferimento rispetto al quale leggere ogni mossa. Lo scarto tra questa riga e una mossa contiene la fortuna del tiro, mai il merito della mossa, e la riga non porta quindi alcuna colonna di errore. Su una posizione di bearoff puro, una seconda tabella, sempre <strong>per giocatore</strong> e sempre presente, dadi impostati o meno, porta l'EPC, il pip count, il wastage, il numero medio di lanci e la deviazione standard; queste cinque colonne non migrano mai. Le due tabelle sono impilate e condividono la stessa griglia di colonne: stessi bordi, stessi riferimenti di colonna, una sola colonna di pallini — si leggono come un unico oggetto a due piani. Il badge di regime, l'attribuzione del motore (vi figura anche la profondità dell'ultima valutazione) e la casella <em>Sfida</em> formano una fascia a parte, allineata a destra sopra le tabelle.</p>
<p>Solo la lista delle mosse candidate scorre — anche la riga <em>prima del lancio</em> resta fissata sopra di essa; il resto del pannello (fatti, badge, decisione di cubo) resta sempre visibile, senza alcuna regolazione particolare della dimensione del pannello.</p>
<p>La tabella dei fatti e la decisione sono calcolate da gammonNet, integrato, senza XG né gnubg. Il calcolo segue la posizione senza mai bloccare l'interfaccia: una profondità 0-ply viene mostrata immediatamente a ogni gesto, poi, dopo mezzo secondo di immobilità, una valutazione più profonda (2 ply per impostazione predefinita, regolabile nella scheda <em>gammonNet</em> della configurazione) la sostituisce in background — qualsiasi nuovo gesto annulla questo calcolo di fondo. La profondità mostrata nella fascia dei badge, o all'interno del badge di regime su una posizione di corsa, è sempre quella che ha effettivamente prodotto il numero mostrato, mai quella richiesta; non si ripete su ogni riga, poiché una valutazione in diretta condivide la stessa profondità per tutte le mosse. L'equità delle mosse candidate e della decisione di cubo segue il punteggio della posizione: in money game è espressa in punti, a un punteggio di match in <strong>equità normalizzata</strong> — la stessa scala di XG e GNU Backgammon, in cui vincere il valore del cubo corrente vale +1 e perderlo −1 — mai mescolate in una stessa tabella. L'intestazione della colonna lo dichiara esplicitamente invece di lasciare indovinare la scala: «Equity (money)» in money game, «Equity (match)» a un punteggio di match. Tiene conto del <strong>cubo vivo</strong>: la ricerca valorizza ogni posizione finale con il modello di cubo (Janowski, efficienza misurata) nello stato del cubo della posizione, come fanno XG e GNU Backgammon nella valutazione <em>cubeful</em>. È ciò che rende visibili al punteggio gli effetti gammon-go e gammon-save — a 4-away/2-away, il giocatore in svantaggio gioca 8/2 6/2 su un 6-4 di apertura perché il suo raddoppio anticipato darà al gammon il valore del match, cosa che una valutazione senza cubo non può vedere. La riga <em>prima del lancio</em>, invece, resta un'equità <strong>cubeless</strong>: è un fatto della posizione, non una decisione. Questo pannello non modifica mai il database: è un calcolo, non un'analisi registrata. Cliccare una mossa candidata la mostra sul tavoliere sotto forma di frecce, esattamente come nel pannello Analisi. Il discreto pulsante <strong>?</strong>, nella fascia dei badge, conduce al repository del motore <code>gammonNet &lt;https://github.com/kevung/gammonNet&gt;</code>_; l'attribuzione completa (rete Strehl, configurazione gammonNet) figura nei Ringraziamenti dell'aiuto.</p>
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
<p>Un terzo distintivo, <strong>Cubo max</strong>, compare quando l'identificatore di origine limita il cubo — sia a un punteggio di incontro sia nel money game. Quello non descrive il calcolo mostrato sopra: il valutatore integrato non modella un tetto, quindi il verdetto è quello di un cubo libero. È proprio per questo che il distintivo c'è: un cubo limitato è l'unica ragione visibile per cui blunderDB ed eXtreme Gammon possono annunciare due verdetti diversi sulla stessa posizione.</p>
<p>Il badge di regime, la profondità di valutazione, il collegamento al motore e la casella <em>Sfida</em> formano una fascia a parte, allineata a destra sopra le tabelle.</p>
<p>Il <strong>giocatore al tiro</strong> e la <strong>posizione del cubo</strong> si modificano direttamente sul tavoliere, come in modalità di modifica: cliccare il rettangolo bearoff/punteggio di un giocatore gli assegna il tiro; cliccare il cubo lo fa ruotare centrato → posseduto in basso → posseduto in alto (clic destro in senso inverso). Il valore del cubo resta fissato — in money game le equità sono espresse in unità del cubo corrente, conta solo il suo proprietario. L'analisi viene ricalcolata immediatamente. In regime stimato, il badge stesso è cliccabile e apre direttamente la scheda <em>Bearoff</em> della configurazione; il suo tooltip spiega perché (verdetto di cubo non stimabile, <code>ADR-0009 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0009-race-win-chances-are-read-or-convolved-cube-verdicts-are-never-estimated.md&gt;</code>__) e come estendere il dominio esatto.</p>
<p>Anche il <strong>punteggio</strong> si modifica direttamente sul tavoliere, come in modalità di modifica: il clic sinistro sul rettangolo del punteggio di un giocatore decrementa il suo numero di punti da fare, il clic destro lo incrementa. Uscire dal punteggio <em>money</em> (-1, -1) modificando un solo campo allinea automaticamente l'altro campo sullo stesso valore anziché lasciare un punteggio incoerente. Su una posizione di bearoff in regime <em>esatto</em>, passare da un punteggio money a un punteggio di match lascia la probabilità di vittoria così com'è (una lettura dal database, valida qualunque sia il riferimento) ma fa passare l'equità e il verdetto di cubo mostrati a quelli del regime <em>valutato</em> — essendo la tabella esatta money per costruzione, non sa rispondere alla domanda posta al punteggio. Il badge diventa allora composito (« esatto (vittoria) · valutato (cubo) ») per dirlo esplicitamente.</p>
<p>I <strong>dadi</strong>, infine, si modificano allo stesso modo, e sono loro a decidere la domanda posta: dadi impostati fanno una decisione di pedine (l'elenco delle mosse candidate), nessun dado una decisione di cubo. Un clic sinistro su un dado ne aumenta il valore (il 6 torna a 1), un clic destro lo diminuisce (l'1 torna a 6); cliccare un dado su un tavoliere che non ne ha ne imposta due in un colpo solo — un dado singolo non sarebbe né una decisione di pedine né una decisione di cubo. Cliccare il rettangolo di un giocatore toglie i dadi per porre una domanda di cubo, e il clic successivo su un dado li rimette com'erano.</p>
<p><em>BACKSPACE</em>, o un doppio clic fuori dal tavoliere, cancella la posizione: tavoliere vuoto, punteggio money (-1, -1), nessun dado impostato — valori propri del pannello Eval, diversi da quelli usati in modalità di modifica (7 ovunque, dadi 3-1), per restare coerenti con ciò che il pannello mostra per impostazione predefinita.</p>
<h4>Matrice del cubo</h4>
<p>Una decisione di cubo non è una proprietà della dama. Le stesse pedine, lo stesso conteggio dei pip, si raddoppiano a 2-away/4-away e non si raddoppiano a 4-away/2-away; chi ha imparato la risposta money ha imparato una sola casella di una griglia. Il pannello Eval mostra la casella che la posizione porta; la <strong>matrice del cubo</strong> mostra l'intera griglia.</p>
<p>Il comando <code>cm</code> la apre sulla posizione visualizzata. Ogni casella dà il verdetto a un punteggio: la riga è il numero di punti che restano da fare al giocatore di turno, la colonna quelli dell'avversario. I quattro verdetti si scrivono <em>ND</em> (niente raddoppio), <em>DP</em> (raddoppio, presa), <em>DR</em> (raddoppio, rifiuto) e <em>TB</em> (troppo buono); una casella rifiutata dal motore porta un punto interrogativo e spiega perché al passaggio del mouse, che dà anche le tre equità della casella. Sono proposte tre lunghezze di incontro: 5, 7 e 9 punti.</p>
<p>Il punteggio della posizione è sostituito da quello di ogni casella; il suo <strong>cubo</strong> è conservato. La griglia risponde a quale punteggio girerei <em>questo</em> cubo, non a ciò che farebbe una posizione centrata. È post-Crawford da un capo all'altro: durante la partita Crawford il cubo non è in gioco, e una colonna di «non potete raddoppiare» non direbbe nulla sulla posizione.</p>
<p>Ogni casella è una ricerca a sé. Il motore tiene conto del punteggio — non gioca la stessa partita a 2-away e a 7-away — quindi una sola ricerca riletta attraverso equità di incontro diverse sarebbe falsa esattamente dove il punteggio conta. La griglia arriva prima in 0-ply, poi si ricalcola alla profondità di visualizzazione configurata una volta che la finestra è a riposo: la stessa escalation del resto del pannello, per una griglia da 9 punti che costa circa un secondo e mezzo.</p>
<p>La stessa griglia si calcola fuori dall'interfaccia, con il comando cubematrix della riga di comando.</p>
<h4>Portare una posizione nel pannello Eval</h4>
<p>Il pannello si apre per impostazione predefinita su una posizione di bearoff, ma lo studio parte il più delle volte da una posizione già in mano. Due gesti ve la portano:</p>
<ul>
<li><strong>Clic destro sul tavoliere</strong>, in un pannello di analisi o durante la navigazione di una partita, poi <em>Valuta questa posizione</em>: il pannello Eval si apre direttamente su questa posizione, così com'è visualizzata. Il menu contestuale non compare nel pannello Eval né nel pannello Ricerca, dove il pulsante destro serve già a posizionare le pedine dell'altro colore.</li>
<li><strong>CTRL-C poi CTRL-V</strong>: copiare la posizione dal pannello di analisi, poi incollarla una volta nel pannello Eval. L'incollaggio accetta anche un identificatore proveniente da altrove — un XGID (eXtreme Gammon, GNU Backgammon, un'altra istanza di blunderDB) o un OGID (OpenGammon): basta che sia negli appunti.</li>
<li><strong>Il comando</strong> <code>import XGID=…</code> (o <code>import OGID=…</code>) per il caso in cui l'identificatore non è negli appunti ma in un messaggio, su un forum letto in un terminale, o prodotto da uno script. È lo stesso verbo di <code>import</code> da solo: senza argomento apre un selettore di file, con un argomento legge l'identificatore. Il percorso è poi identico a quello dell'incollaggio — stessa lettura, stessa deduplicazione, stessa apertura della posizione importata.</li>
</ul>
<p>Un OGID porta solo una posizione: né valutazione, né commento. La posizione arriva quindi senza analisi, esattamente come un XGID nudo, e il valutatore integrato può colmare il vuoto in seguito.</p>
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
<p><strong>Probabilità di vittoria e verdetto, regime valutato.</strong> Fuori dal dominio esatto, la probabilità di vittoria proviene dall'output grezzo di gammonNet (ricerca a 0 o 2 ply a seconda del gesto, mai letta in una tabella), e il verdetto da un « Decide » Janowski applicato a tale output — la ricerca <em>gioca</em> la traiettoria invece di riassumerne un'istantanea, che è precisamente ciò che il regime stimato non poteva fare (vedere più sotto) e consente, unico dei tre regimi insieme all'esatto, un verdetto <strong>al punteggio di match</strong>.</p>
<p>Questo regime è stato misurato, non soltanto supposto, rispetto alla tabella two-sided integrata (<code>TestEvalMeasure</code>, 4000 decisioni money campionate, parametri canonici 2 ply k=12): accordo del verdetto money <strong>93,4 %</strong> (3735/4000), ripartito per distanza dal punto di presa di gammonNet — 61,1 % a meno dell'1 % dal punto di presa (la zona più sensibile a un testa o croce), 88,3 % tra l'1 e il 5 %, 91,5 % tra il 5 e il 10 %, 94,0 % tra il 10 e il 20 %, 94,4 % oltre. Scarto di probabilità di vittoria: media 0,85 %, mediana 0,44 %, 95º percentile 3,21 %, massimo 8,30 %. Scarto di equità cubeful: media 0,039, mediana 0,018, 95º percentile 0,151, massimo 0,406. La forma è quella attesa: l'essenziale del disaccordo si concentra esattamente al punto di presa, dove due metodi legittimamente diversi divergono di più su una decisione serrata — non un errore diffuso che costerebbe equità ovunque.</p>
<p>Questa misura riguarda decisioni <strong>money</strong>, in corsa. Il verdetto al punteggio di match — che solo questo regime sa rendere — e le posizioni di contatto non hanno una misura pubblicata: quanto precede non si trasferisce a questi casi.</p>
<p><strong>Perché non più profondo di 2 ply?</strong> Perché la misura dice che non rende nulla. Una decisione di pedine costa 99 ms a 2 ply e 8,4 s a 3 ply sulla stessa macchina — <strong>ottantacinque volte di più</strong>. Su quaranta decisioni reali rigiocate a entrambe le profondità, la ricerca più profonda ha cambiato idea <strong>due volte</strong>, e in entrambi i casi il guadagno che si attribuiva valeva al massimo 0,0005 di equità normalizzata: due ordini di grandezza sotto 0,020, la soglia a partire dalla quale eXtreme Gammon parla di errore. Per decisione, tutti i casi insieme, il guadagno è 0,0000.</p>
<p>L'impostazione non è dunque proposta. Non si tratta di dire che 3 ply non valga nulla in generale, ma che su <em>questa</em> rete, con il filtro canonico, non paga l'attesa di chi sta davanti a un pannello. La misura è riproducibile (<code>TestThreePlyMeasure</code>) e la conclusione sarà rivalutata se la rete cambia.</p>
<p><strong>Perché il verdetto stimato non esiste?</strong> Quanto segue riguarda specificamente il metodo per <em>convoluzione</em> (regime stimato), non il regime valutato descritto sopra: l'equità cubeful è un problema di <em>traiettoria</em> (quando raddoppiare), che nessun riassunto statistico della posizione riesce a catturare — il miglior modello statico misurato lascia un errore residuo (deviazione standard 0,016 di equità, massimo 0,20) sufficiente a invertire tutte le decisioni serrate. Allo stesso modo, la conversione del verdetto al punteggio del match tramite una tabella di equità di match è risultata insufficiente (12 % di disaccordi con l'analisi 2-ply di GNUbg, con veri blunder). Poiché un verdetto sbagliato mostrato con sicurezza è peggio di nessun verdetto, la convoluzione non ha mai avuto il diritto di mostrare un verdetto — è una ricerca che gioca la traiettoria, non un riassunto statistico, a colmare questa lacuna.</p>
<div class="admonition note">
<p>Le basi di bearoff sono tabelle matematiche immutabili. blunderDB le calcola da sé, identiche allo strumento <code>makebearoff</code> di GNUbg — byte per byte — nella scheda <em>Bearoff</em> della configurazione o con <code>blunderdb bearoff generate</code>.</p>
</div>
<h3>Pannello Anki</h3>
<p>Il pannello <strong>Anki</strong> (<em>CTRL-K</em>) permette di studiare le posizioni tramite ripetizione dilazionata utilizzando l'algoritmo FSRS. L'utente può creare mazzi a partire da raccolte o risultati di ricerca.</p>
<p><strong>Creazione di mazzi:</strong> Cliccare su <em>New Deck</em> per creare un mazzo a partire da una raccolta o dai risultati di ricerca correnti. I mazzi basati su una ricerca si sincronizzano automaticamente all'apertura della scheda Anki.</p>
<p><strong>Revisione:</strong> Selezionare un mazzo e poi cliccare su <em>Study</em> (oppure fare doppio clic su un mazzo) per iniziare la revisione delle carte in scadenza. Ogni carta mostra la posizione corrispondente sul board. Valutare il proprio richiamo con i tasti <em>1</em> (Da rivedere), <em>2</em> (Difficile), <em>3</em> (Bene) o <em>4</em> (Facile). Premere <em>Esc</em> per interrompere e tornare all'elenco dei mazzi.</p>
<p><strong>Le decisioni di cubo fanno due carte, concatenate.</strong> Una decisione di cubo è due domande — «raddoppio?», poi «presa?» — e blunderDB le registra da sempre come due posizioni. Un mazzo che ne seleziona una sola metà riceve l'altra: la decisione è completata, non ampliata. E quando entrambe sono dovute, la seconda arriva <strong>immediatamente</strong> dopo la prima.</p>
<p>Ciascuna conserva il proprio voto e il proprio calendario: non sono due tempi di una stessa carta, sono due carte. La concatenazione non anticipa alcuna scadenza — ordina le carte già dovute, nulla di più. Nascendo insieme, sono dovute insieme la prima volta, ed è lì che serve.</p>
<p><strong>Mostrare la risposta:</strong> La carta pone una domanda — quale mossa giocare, o quale azione di cubo. Riflettere, poi premere <em>SPAZIO</em> (o cliccare sulla zona nascosta) per svelare la risposta: l'analisi registrata della posizione, così come la presenta la scheda Analisi. Appare sotto i pulsanti di valutazione, che restano al loro posto e a portata di mano. Cliccare su una mossa dell'elenco la mostra sul tavoliere.</p>
<p>Nulla obbliga a svelare la risposta per valutare: se si è sicuri, i tasti da <em>1</em> a <em>4</em> restano attivi. La risposta viene nascosta di nuovo alla carta successiva, ma non se si cambia semplicemente scheda — si vada pure a consultare il pannello Eval o il commento della posizione, la si ritroverà al ritorno.</p>
<p>Una posizione priva di analisi registrata lo indica direttamente, senza zona nascosta.</p>
<p><strong>Limitare la sessione.</strong> Per impostazione predefinita una sessione di ripasso arriva fino in fondo alle carte in scadenza. Puoi limitarla a un numero di carte, per mazzo, nelle Impostazioni: spunta <em>Limita la sessione</em> e indica quante carte deve servire una sessione. Quando il limite è raggiunto, la sessione si ferma dicendolo — il messaggio distingue «limite raggiunto, ancora tante carte in scadenza» da una coda davvero esaurita. Per continuare comunque c'è l'allenamento libero: propone altre posizioni senza modificare nulla della pianificazione.</p>
<p>Un limite di <strong>0</strong> non serve nessuna carta: è uno stato a pieno titolo, utile per congelare un mazzo il tempo di preparare un torneo, e non è la stessa cosa di «nessun limite». Il pulsante <em>Study</em> è allora inattivo.</p>
<p>Il limite riguarda la <strong>sessione</strong>, non la giornata. Un mazzo di blunderDB è costruito su una raccolta o su una ricerca: è un corpus finito, introdotto in poche sessioni, il cui volume quotidiano è già limitato dalla sua dimensione. Un tetto giornaliero non morderebbe mai, oppure creerebbe un arretrato su un mazzo che stava in una sessione.</p>
<p><strong>Allenamento libero (cram):</strong> Il pulsante <em>Cram</em>, accanto a <em>Study</em>, avvia una sessione di allenamento libero: ti vengono mostrate posizioni casuali del mazzo senza tenere conto della pianificazione FSRS. Questa modalità <strong>non modifica mai il piano di ripetizione dilazionata</strong> — ideale per scaldarsi prima di un torneo o per ripassare intensamente un mazzo tematico senza alterarne l'ordine. Un'etichetta <em>Cram</em> sostituisce lo stato della carta e un pulsante <em>Avanti</em> (tasti *1*–*4*) scorre le posizioni. <em>Esc</em> torna all'elenco senza salvare una sessione interrotta.</p>
<p><strong>Mettere da parte una carta, senza votarla.</strong> Durante un ripasso, un clic destro sull'intestazione della carta offre tre gesti che la tolgono dalla sessione senza dire nulla allo scheduler:</p>
<ul>
<li><strong>Sospendere</strong> — la carta conserva il suo calendario e non risale finché è sospesa. È il modo di mettere da parte una carta sbagliata, o non ancora utile, senza perdere lo storico che vi è legato.</li>
<li><strong>Rinviare</strong> — la carta sparisce fino all'indomani. A differenza della sospensione, questo non dice nulla sul suo valore: è per quella appena vista altrove, o che si preferisce non incrociare due volte in una serata.</li>
<li><strong>Togliere</strong> — la carta lascia il mazzo, previa conferma. La posizione resta nella base: un mazzo è una lista di studio sulla biblioteca, mai una sua copia.</li>
</ul>
<p>Nessuno di questi tre gesti registra un voto: una carta messa da parte non è una carta risposta, e non conta nel totale della sessione.</p>
<p><strong>Registro dei ripassi.</strong> Nelle Impostazioni di un mazzo, il pulsante <em>Registro dei ripassi</em> mostra ciò che è stato <strong>detto</strong> allo scheduler — data, posizione, voto, stato, intervallo concesso — in contrasto con ciò che prevede. È l'unico posto dove si vede un voto inserito per errore. Lì non si corregge: il calendario resta fuori portata, e proprio questa regola rende utile il registro — il passato non si riscrive, ma si può conoscere.</p>
<p><strong>Interruzione/Ripresa:</strong> È possibile interrompere una sessione di revisione in qualsiasi momento con <em>Esc</em>. Il pulsante cambia in <em>Resume</em> e mostra la tua progressione. Cliccarci sopra per riprendere da dove ci si era fermati.</p>
<p><strong>Gestione dei mazzi:</strong> Usare i pulsanti d'azione per rinominare, sincronizzare, reimpostare o eliminare i mazzi (viene chiesta conferma per queste ultime due azioni). I parametri FSRS (ritenzione obiettivo, intervallo massimo, casualità) si possono configurare per mazzo nelle Impostazioni (icona a ingranaggio).</p>
<p><strong>Ritenzione: l'obiettivo e la misura.</strong> La <em>ritenzione obiettivo</em> è la tua scelta sul compromesso fra carico di lavoro e qualità del richiamo: più è alta, più gli intervalli si accorciano e più ripassi. Accanto, le Impostazioni mostrano la <strong>ritenzione misurata</strong> sui tuoi stessi ripassi — un'informazione, mai un comando: blunderDB non modifica il tuo obiettivo per inseguire il tuo tasso di successo. Sotto una ventina di ripassi la misura non viene mostrata: si leggerebbe come un fatto mentre è solo rumore.</p>
<p>Cambiare la ritenzione <strong>non è retroattivo</strong>: ogni carta adotta il nuovo ritmo al ripasso successivo, e le scadenze già fissate non si spostano. L'effetto è quindi graduale, e invisibile il giorno stesso.</p>
<p>L'<em>intervallo massimo</em> limita la spaziatura. Un mazzo creato di recente parte da un anno: una posizione che l'algoritmo rimanderebbe di diversi anni ha lasciato il mazzo senza che tu l'abbia deciso, e il tuo stesso gioco cambia più in fretta di così. I mazzi più vecchi conservano il valore che avevano.</p>
<h3>Micro-allenamenti</h3>
<p>Il pannello Anki fa ripassare un <strong>giudizio</strong>; i micro-allenamenti fanno lavorare i tre <strong>calcoli</strong> che si fanno al tavolo, sotto l'orologio, e che nessuna ripetizione dilazionata sviluppa. Il comando <code>train</code> avvia una sessione di cinque domande:</p>
<ul>
<li><code>train pips</code> — contare i pip del giocatore di turno, sulla posizione mostrata.</li>
<li><code>train epc</code> — stimare l'EPC dello stesso giocatore, su una posizione di corsa che il motore sa valutare.</li>
<li><code>train tp</code> — ritrovare il punto di presa di una corsa lunga a un punteggio estratto a caso, quello della tabella <code>tp2_live</code>.</li>
</ul>
<p>La domanda È la posizione mostrata: la tavola è quella dell'applicazione, e la barra sopra porta solo la domanda, l'inserimento e la correzione. La risposta si digita e si convalida da tastiera (<em>Invio</em> verifica, poi passa alla successiva; <em>Esc</em> lascia la sessione).</p>
<p>La tolleranza dipende dall'esercizio, ed è dichiarata anziché indovinata: il conteggio dei pip non ne ha <strong>nessuna</strong> — un'addizione giusta a un pip di distanza è un'addizione sbagliata — l'EPC accetta mezzo pip, il punto di presa due punti percentuali. Alla fine, la sessione mostra il numero di risposte giuste e il tempo <strong>mediano</strong> per domanda.</p>
<p>Solo questo riepilogo viene conservato, nei metadati della base: la sessione non tiene traccia domanda per domanda, e nulla viene scritto finché non è terminata. Uscire a metà strada quindi non registra nulla.</p>
<h4>Quiz: il PR di allenamento</h4>
<p><code>train quiz</code> pone un quarto tipo di domanda. Il pannello Anki fa memorizzare; il quiz <strong>mette alla prova</strong>. Cinque posizioni già analizzate vengono estratte dalla lista percorsa, e occorre decidere:</p>
<ul>
<li>su una decisione di pedine, scrivere la mossa da tastiera, in notazione (<code>13/7 8/7</code>);</li>
<li>su una decisione di cubo, cliccare <em>Nessun raddoppio</em>, <em>Raddoppio, presa</em> o <em>Raddoppio, passo</em>.</li>
</ul>
<p>Il pannello Analisi resta mascherato finché la domanda non ha una risposta: porta la risposta, e una domanda la cui risposta è mostrata accanto non è una domanda.</p>
<p>La correzione distingue tre esiti, e confonderli mentirebbe. Una <strong>mossa illegale</strong> non è una mossa mal scelta: è un errore di regole. Una <strong>mossa legale che il motore non ha classificato</strong> non è affatto un errore: semplicemente non ha prezzo, e non costa nulla alla sessione. Una mossa classificata costa quello che l'analisi dice, in millipunti.</p>
<p>Alla fine, la sessione mostra un <strong>PR di quiz</strong> calcolato con la formula che le statistiche applicano al gioco reale — 500 × errore medio in equità normalizzata. È ciò che rende i due numeri confrontabili: un PR di quiz di 6 e un PR di incontro di 6 misurano la stessa cosa sulla stessa scala.</p>
<h3>Pannello Metadati</h3>
<p>Il pannello <strong>Metadati</strong> visualizza le informazioni generali del database corrente: nome, descrizione, numero di posizioni, numero di match e di partite, versione dello schema. Accessibile tramite il comando <code>meta</code>.</p>
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
<p>I campi <em>Utente</em>, <em>Descrizione</em> e <em>Data</em> descrivono il <strong>file prodotto</strong>; sono precompilati a partire dal database di origine. La casella <em>I miei filtri salvati</em> è a sé stante: non esporta contenuti ma le tue ricerche salvate, inutili nel database di qualcun altro.</p>
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
<li>«✓ firma verificata — contrassegnato da te»: il file reca il tuo contrassegno, intatto;</li>
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
<td>Reimposta board, cubo, punteggio e dadi.</td>
</tr>
<tr>
<td>CTRL-G</td>
<td>Mostra i metadati della posizione.</td>
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
<td>PageUp, h</td>
<td>Prima posizione / Partita precedente (navigazione match).</td>
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
<td>PageDown, l</td>
<td>Ultima posizione / Partita successiva (navigazione match).</td>
</tr>
<tr>
<td>r</td>
<td>Carica una posizione casuale.</td>
</tr>
</tbody>
</table>
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
<td>CTRL-P</td>
<td>Mostra/nascondi i commenti.</td>
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
<td>?</td>
<td>Mostra/nascondi l'aiuto.</td>
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
<td>CTRL-PageUp, MAIUSC-J</td>
<td>Vista precedente.</td>
</tr>
<tr>
<td>CTRL-PageDown, MAIUSC-K</td>
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
<td>Esc</td>
<td>Deseleziona la mossa. Se nessuna mossa è selezionata, chiudi il pannello.</td>
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
<td>SU, k</td>
<td>Seleziona la mossa precedente (quando una mossa è selezionata).</td>
</tr>
<tr>
<td>GIÙ, j</td>
<td>Seleziona la mossa successiva (quando una mossa è selezionata).</td>
</tr>
<tr>
<td>Esc</td>
<td>Deseleziona la mossa.</td>
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
<td>Clic, Doppio clic</td>
<td>Seleziona un torneo (mostrane il dettaglio).</td>
</tr>
<tr>
<td>SU, k</td>
<td>Seleziona il torneo precedente.</td>
</tr>
<tr>
<td>GIÙ, j</td>
<td>Seleziona il torneo successivo.</td>
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
<td>PageUp</td>
<td>Inizio del contenuto.</td>
</tr>
<tr>
<td>PageDown</td>
<td>Fine del contenuto.</td>
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
<td>epc</td>
<td>Apre il pannello Eval (Effective Pip Count, probabilità di vittoria e verdetto di cubo in bearoff). <code>epc</code> è il vecchio nome di questo pannello, conservato.</td>
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
<td>ask</td>
<td>Traduce una frase a parole — francese o inglese — in token di ricerca: <code>ask my cube blunders at a score</code>. I token vengono scritti nella barra dei comandi, non eseguiti: si rileggono, poi Invio. Ciò che non è stato compreso viene detto, mai indovinato.</td>
</tr>
<tr>
<td>like</td>
<td>Sostituisce la lista percorsa con le posizioni più vicine a quella corrente — o a quella il cui indice è indicato (<code>like 42</code>). La vicinanza è una distanza di trasporto in pip di pedina: non è un filtro, ordina l'intera base invece di restringerla, e quindi non si combina con i token di ricerca.</td>
</tr>
<tr>
<td>train</td>
<td>Avvia una sessione di micro-allenamento. Prende un argomento: <code>train pips</code> (conteggio dei pip), <code>train epc</code>, <code>train tp</code> (punto di presa al punteggio), <code>train quiz</code> (la mossa o l'azione di cubo, valutate contro l'analisi registrata). Cinque domande, cronometrate, corrette sul momento.</td>
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
<td>[number]</td>
<td>Vai alla posizione con l'indice indicato.</td>
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
<td>#tag1 tag2 ...</td>
<td>Etichetta la posizione corrente.</td>
</tr>
<tr>
<td>e</td>
<td>Carica tutte le posizioni del database.</td>
</tr>
<tr>
<td>blunders, bl [n]</td>
<td>Carica gli errori peggiori (equity/MWC) nella vista di analisi, secondo il filtro statistico corrente. Un numero opzionale sceglie quanti caricarne (<code>bl 50</code>); 10 per impostazione predefinita.Carica gli errori peggiori (equity/MWC) nella vista di analisi, secondo il filtro statistico corrente.</td>
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
<td>Cerca tra le posizioni attualmente filtrate.</td>
</tr>
</tbody>
</table>
<h3>Filtri di ricerca</h3>
<p>Questa tabella è il riferimento della grammatica di ricerca: la riga di comando, la biblioteca di filtri e l'opzione <code>--query</code> di <code>blunderdb search</code> leggono tutte gli stessi token. La colonna <em>Equivalente CLI</em> dà, quando esiste, l'opzione di <code>search</code> che fa la stessa cosa (vedere Interfaccia a riga di comando (CLI)); un trattino segnala un filtro che solo la grammatica esprime.</p>
<p>Cinque token non portano il proprio valore: lo leggono sul tavoliere di ricerca. <code>cube</code> e <code>score</code> riprendono il cubo e il punteggio lì impostati, <code>d</code> il tipo di decisione, <code>D</code> e <code>D1</code> i dadi, <code>x</code> la struttura disegnata nella scheda <em>Tranne</em>. Un lancio non si scrive quindi mai nel token: <code>D65</code> non esiste, solo la forma di esclusione porta le sue cifre (<code>xD65</code>). Sulla riga di comando, dove non c'è tavoliere, questi token si confrontano con un tavoliere vuoto; sono le opzioni della terza colonna che occorre impiegare al loro posto.</p>
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
<td>La posizione è stata incontrata più di x volte nella base — il numero di mosse che vi arrivano, in tutti gli incontri. Forme <code>n&gt;3</code>, <code>n&lt;2</code>, <code>n3,10</code> e <code>n4</code> (esattamente quattro).</td>
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
<td>Data di aggiunta della posizione dopo x (AAAA/MM/GG).</td>
<td>—</td>
</tr>
<tr>
<td>T&lt;x</td>
<td>Data di aggiunta della posizione prima di x (AAAA/MM/GG).</td>
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
<td>Cerca posizioni di una partita a cui ha partecipato il giocatore indicato, su entrambi i lati (es. <code>pl'Alice'</code>). Non distingue maiuscole e minuscole.</td>
<td>—</td>
</tr>
</tbody>
</table>
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
        La rete neurale <strong>strehl-prob5-512-512-256-128</strong> è opera di <strong>Alexander Strehl</strong> (<em>alexstrehl/backgammon-ai-engine</em>, MIT). La ricerca, il modello di cubo e la
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
        I database di bearoff a un lato (6 punti, 15 pedine, per l'EPC) e a due lati (6 punti, 6 pedine, per i verdetti di cubo nelle corse) sono stati generati con
        <strong>GNU Backgammon</strong> (GNUbg). GNUbg è software libero sotto licenza GPL; queste tabelle sono dati da esso prodotti, accreditati come tali.
    </li>
    <li>I file di match sono letti da <em>xgparser</em>, <em>gnubgparser</em> e <em>bgfparser</em> (MIT).</li>
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
