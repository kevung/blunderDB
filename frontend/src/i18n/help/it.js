// GENERATED FILE — do not edit by hand, and do not translate it here.
//
// Produced by `go run ./cmd/help-gen` (make help) from:
//   - doc/source/raccourcis.rst  → the "shortcuts" tab
//   - doc/source/cmd_mode.rst    → the "commands" tab
//   - doc/source/locale/<lang>/LC_MESSAGES/*.po for the eight translations
//   - frontend/src/i18n/help/prose/<lang>.html → the "manual" and "about" tabs
//
// Fix the documentation (and its .po catalogues), or the prose fragment, then
// run `make help`. TestHelpBundlesAreCurrent fails if this file is stale.
export default {
    manual: `
<h3>Introduzione</h3>
<p>
    blunderDB è un software per creare database di posizioni di backgammon. Il suo punto di forza principale è offrire un unico luogo in cui raccogliere le posizioni che un giocatore ha incontrato
    (online, nei tornei) e poter ristudiare queste posizioni filtrandole secondo vari filtri combinabili arbitrariamente. blunderDB può anche essere usato per creare cataloghi di posizioni di
    riferimento.
</p>
<p>Le posizioni sono memorizzate in un database rappresentato da un file .db.</p>

<h3>Interazioni principali</h3>
<p>Le principali interazioni possibili con blunderDB sono:</p>
<ul>
    <li>aggiungere una nuova posizione,</li>
    <li>modificare una posizione esistente,</li>
    <li>copiare il tavoliere come immagine PNG negli appunti (<strong>Ctrl+X</strong>), oppure il tavoliere con la sua analisi (<strong>Ctrl+X, Ctrl+X</strong>),</li>
    <li>eliminare una posizione esistente,</li>
    <li>cercare una o più posizioni,</li>
    <li>importare match da varie fonti (XG, GNUbg, BGBlitz, Jellyfish), inclusi i commenti dei file XG,</li>
    <li>scorrere le mosse di un match importato,</li>
    <li>organizzare le posizioni in collezioni,</li>
    <li>organizzare i match in tornei,</li>
    <li>analizzare in blocco, da terminale, le posizioni prive di analisi grazie al valutatore gammonNet integrato (comando <strong>analyze</strong> di blunderDB).</li>
</ul>
<p>L'utente può etichettare liberamente le posizioni e annotarle con commenti.</p>

<h3>Descrizione dell'interfaccia</h3>
<p>L'interfaccia di blunderDB è strutturata dall'alto verso il basso come segue:</p>
<ul>
    <li>[in alto] la barra degli strumenti, che raccoglie tutte le operazioni principali eseguibili sul database,</li>
    <li>[al centro] l'area di visualizzazione principale, che permette di mostrare o modificare posizioni di backgammon,</li>
    <li>[in basso] la barra di stato, che integra la riga di comando e presenta varie informazioni sulla posizione corrente.</li>
</ul>
<p>È possibile mostrare pannelli per:</p>
<ul>
    <li>mostrare i dati di analisi associati alla posizione corrente (da XG, GNUbg o BGBlitz),</li>
    <li>mostrare, aggiungere o modificare commenti,</li>
    <li>scorrere i match importati e navigare tra le loro mosse (pannello Match),</li>
    <li>gestire collezioni di posizioni (pannello Collezione),</li>
    <li>studiare posizioni con la ripetizione dilazionata (pannello Anki),</li>
    <li>gestire i tornei (pannello Torneo),</li>
    <li>mostrare statistiche di rendimento (pannello Stats),</li>
    <li>valutare qualsiasi posizione con il motore integrato, e calcolare l'EPC di una posizione di bearoff (pannello Eval),</li>
    <li>consultare i filtri di ricerca salvati (pannello Libreria filtri),</li>
    <li>consultare la cronologia delle ricerche (pannello Cronologia ricerche).</li>
</ul>
<p>L'area di visualizzazione principale offre all'utente:</p>
<ul>
    <li>un tavoliere per mostrare o modificare una posizione di backgammon,</li>
    <li>il livello e il proprietario del cube,</li>
    <li>il pip count di ciascun giocatore,</li>
    <li>il punteggio di ciascun giocatore,</li>
    <li>i dadi da giocare. Se sui dadi non viene mostrato alcun valore, la posizione dei dadi indica quale giocatore è di turno e che la posizione è una decisione sul cube.</li>
</ul>
<p>La barra di stato mostra da sinistra a destra:</p>
<ul>
    <li>la riga di comando (premi <strong>Spazio</strong> per aprirla),</li>
    <li>un messaggio informativo relativo all'ultima operazione eseguita,</li>
    <li>l'indice della posizione corrente, seguito dal numero totale di posizioni (o informazioni su mossa/partita durante la navigazione di un match).</li>
</ul>
<p>Nel caso di posizioni risultanti da una ricerca dell'utente, il numero di posizioni indicato nella barra di stato corrisponde al numero di posizioni filtrate.</p>

<h3>Navigare tra le posizioni</h3>
<p>Per impostazione predefinita, blunderDB ti permette di:</p>
<ul>
    <li>scorrere le diverse posizioni della libreria corrente,</li>
    <li>mostrare le informazioni di analisi associate a una posizione,</li>
    <li>mostrare, aggiungere e modificare commenti su una posizione.</li>
</ul>

<h3>Modificare le posizioni</h3>
<p>
    Premere il tasto <strong>Tab</strong> apre il pannello di ricerca e permette di modificare una posizione sul tavoliere per aggiungerla al database o per definire una struttura di posizione da
    cercare. La distribuzione delle pedine, il cube, il punteggio e il turno possono essere modificati con il mouse.
</p>

<h3>Riga di comando</h3>
<p>
    La riga di comando, integrata nella barra di stato, permette di eseguire tutte le funzionalità di blunderDB: operazioni sul database, navigazione tra le posizioni, mostrare analisi e commenti,
    cercare posizioni con filtri... Dopo aver preso confidenza con l'interfaccia, è consigliabile usare progressivamente la riga di comando, che permette un uso potente e fluido di blunderDB,
    soprattutto per le funzionalità di ricerca delle posizioni.
</p>
<p>
    Per aprire la riga di comando, premi il tasto <strong>Spazio</strong>. Appare un prompt nella barra di stato. Digita il tuo comando e premi <strong>Enter</strong> per eseguirlo. Premi
    <strong>Escape</strong>
    per annullare.
</p>
<p>
    blunderDB esegue le query inviate dall'utente purché siano valide e modifica immediatamente lo stato del database se necessario. Non sono richieste azioni di salvataggio esplicite da parte
    dell'utente.
</p>
<p>
    Per affinare una ricerca all'interno di posizioni già filtrate, usa il comando <strong>ss</strong> seguito da filtri (es. <strong>ss nc</strong>). Questo restringe la ricerca alle sole posizioni
    attualmente mostrate, permettendo di restringere progressivamente i risultati. Il pannello di ricerca (<strong>Ctrl+F</strong>) offre anche una casella "Cerca nei risultati correnti" per la stessa
    funzionalità.
</p>

<h3>Pannello Eval</h3>
<p>
    Il pannello <strong>Eval</strong> valuta qualunque posizione si trovi sul tavoliere: probabilità di vittoria, gammon e backgammon, equity, mosse candidate ordinate e l'unica decisione che la
    posizione richiede — giocare una mossa o raddoppiare. Il calcolo è fatto da gammonNet, integrato: non servono né eXtreme Gammon né GNU Backgammon.
</p>
<p>
    Per aprirlo premere <strong>Ctrl+E</strong>, fare clic sulla scheda Eval del pannello inferiore oppure digitare <strong>epc</strong> nella riga di comando. Il tavoliere si apre su una
    configurazione di bearoff standard (15 pedine), a meno che non vi sia stata inviata una posizione dal database. Le pedine si aggiungono e si tolgono liberamente con il mouse; la valutazione segue
    ogni modifica.
</p>
<p>
    Su una posizione di bearoff il pannello <strong>si specializza</strong>: una seconda tabella, per giocatore, riporta l'EPC (Effective Pip Count) calcolato dal database di bearoff unilaterale a 6
    punti di GNUbg —
</p>
<ul>
    <li><strong>EPC</strong>: il numero medio di pip necessari per portare fuori tutte le pedine,</li>
    <li><strong>Pip Count</strong>: il pip count grezzo,</li>
    <li><strong>Wastage</strong>: la differenza tra EPC e pip count,</li>
    <li><strong>Avg Rolls</strong>: il numero medio di lanci per portare fuori tutte le pedine,</li>
    <li><strong>Std Dev</strong>: la deviazione standard di quel numero di lanci.</li>
</ul>
<p>Quando entrambi i giocatori hanno pedine nella propria casa, una sezione di confronto mostra le differenze di EPC e di pip count.</p>
<p>
    Su una corsa pura, un'ulteriore tabella mostra le probabilità di vittoria dei due giocatori e, quando la posizione è coperta da una tabella two-sided (tabella a 6 pedine per giocatore calcolata al
    primo avvio, tabella estesa a 11 pedine calcolata dalla scheda Bearoff della configurazione), le equity money esatte e la migliore decisione di cubo. Fuori da quel dominio la probabilità di
    vittoria è stimata (badge «stimato» con il suo margine d'errore) e non viene mostrata alcuna decisione. Il giocatore di turno si modifica facendo clic sul rettangolo uscita/punteggio di un
    giocatore, la posizione del cubo facendo clic sul cubo del tavoliere.
</p>
<p>
    La casella <strong>Sfida</strong> nasconde i risultati a ogni modifica della posizione; fare clic su un'area per rivelarla — ideale per allenarsi a stimare un'equity, un EPC o una decisione di
    cubo prima di verificare.
</p>
<p>Per chiudere il pannello Eval, premere di nuovo <strong>Ctrl+E</strong> oppure passare a un'altra scheda.</p>

<h3>Navigazione dei match</h3>
<p>
    blunderDB permette di scorrere le mosse dei match importati. Apri il pannello Match con <strong>Ctrl+Tab</strong> e fai doppio clic su un match (o premi <strong>Enter</strong>) per caricarne le
    posizioni.
</p>
<p>
    Durante la navigazione di un match, l'ultima posizione visitata viene salvata e ripristinata automaticamente. Usa i tasti <strong>Sinistra</strong>/<strong>Destra</strong> per spostarti tra le
    posizioni, e <strong>PageUp</strong>/<strong>PageDown</strong> per saltare tra le partite.
</p>
<p>Il pannello di analisi (<strong>Ctrl+L</strong>) mostra l'analisi di ogni mossa, evidenziando la mossa giocata. Premi <strong>d</strong> per alternare tra analisi delle pedine e del cube.</p>

<h3>Collezioni</h3>
<p>
    Le collezioni permettono di organizzare le posizioni in gruppi personalizzati. Apri il pannello Collezione con <strong>Ctrl+B</strong>, poi fai doppio clic su una collezione per scorrerne le
    posizioni. Le collezioni e le posizioni che contengono possono essere riordinate trascinandole.
</p>

<h3>Anki (ripetizione dilazionata)</h3>
<p>Il pannello Anki (<strong>Ctrl+K</strong>) offre la ripetizione dilazionata per studiare posizioni di backgammon usando l'algoritmo FSRS.</p>
<p>
    <strong>Creare mazzi:</strong> Fai clic su <em>Nuovo mazzo</em> per creare un mazzo a partire da una collezione o dai risultati di ricerca correnti. I mazzi basati su ricerche si sincronizzano
    automaticamente quando si attiva la scheda Anki.
</p>
<p>
    <strong>Ripassare:</strong> Seleziona un mazzo e fai clic su <em>Studia</em> (o fai doppio clic su un mazzo) per iniziare a ripassare le carte in scadenza. Ogni carta mostra la posizione
    corrispondente sul tavoliere. Valuta il tuo ricordo con i tasti <strong>1</strong> (Di nuovo), <strong>2</strong> (Difficile), <strong>3</strong> (Bene) o <strong>4</strong> (Facile). Premi
    <strong>Esc</strong> per fermarti e tornare alla lista dei mazzi.
</p>
<p>
    <strong>Limitare la sessione:</strong> Nelle impostazioni del mazzo puoi limitare una sessione a un numero di carte. La sessione si ferma dicendolo, e l'allenamento libero resta disponibile per
    proseguire senza toccare la pianificazione. Un limite di <em>0</em> non serve alcuna carta: non è la stessa cosa che nessun limite.
</p>
<p>
    <strong>Ritenzione:</strong> La ritenzione desiderata è la tua scelta sul compromesso carico/qualità. Le impostazioni mostrano accanto la ritenzione <em>misurata</em> sui tuoi ripassi:
    un'informazione, mai un pilotaggio. Cambiare l'obiettivo non è retroattivo: ogni carta adotta il nuovo ritmo al ripasso successivo.
</p>
<p>
    <strong>Mostrare la risposta:</strong> La carta pone una domanda; rifletti, poi premi <strong>Spazio</strong> (o fai clic sull'area mascherata) per rivelare l'analisi registrata della posizione.
    Compare sotto i pulsanti di valutazione, che restano a portata. Non è necessario rivelarla per valutare, e si rimaschera alla carta successiva, non quando cambi semplicemente scheda.
</p>
<p>
    <strong>Ferma/Riprendi:</strong> Puoi fermare una sessione di ripasso in qualsiasi momento premendo <strong>Esc</strong>. Il pulsante cambia in <em>Riprendi</em> mostrando i tuoi progressi. Fai
    clic su di esso per continuare da dove avevi lasciato.
</p>
<p>
    <strong>Gestione dei mazzi:</strong> Usa i pulsanti di azione per rinominare, sincronizzare, reimpostare o eliminare i mazzi. I parametri di FSRS (ritenzione obiettivo, intervallo massimo, fuzz)
    possono essere configurati per ogni mazzo nelle Impostazioni (icona a ingranaggio).
</p>

<h3>Tornei</h3>
<p>
    I tornei permettono di raggruppare i match per evento. All'importazione, un match viene classificato nel torneo che il suo file nomina, creato se necessario; un match già classificato non viene
    mai spostato. Apri il pannello Torneo con <strong>Ctrl+Y</strong> per gestire i tornei e assegnare loro i match.
</p>

<h3>Stats</h3>
<p>
    Il pannello Stats (<strong>Ctrl+D</strong>) mostra statistiche di rendimento (PR e costo in MWC) calcolate a partire da tutte le posizioni importate. Usa la barra dei filtri per restringere
    l'analisi per giocatore, torneo, intervallo di date, tipo di decisione o lunghezza del match. Fai clic su un qualsiasi indicatore per approfondire le posizioni corrispondenti. La scheda
    <strong>Giocatori</strong> elenca, per giocatore, il numero di partite, il bilancio, le decisioni, il PR (pedine e cubo), lo Snowie, i blunder e la fortuna misurata sui lanci noti.
</p>

<h3>Marca ed esportazione protetta</h3>
<p>Durante un'esportazione (<strong>export_db</strong> o la finestra Esporta), è possibile attivare liberamente due protezioni indipendenti, l'una, l'altra, o entrambe insieme:</p>
<ul>
    <li>
        <strong>Marca:</strong> marca il file esportato con la sua origine (chi lo ha prodotto, una nota facoltativa). La marca è firmata con la tua identità dell'emittente: non può essere alterata né
        contraffatta a nome di qualcun altro — ma non è incancellabile e non impedisce alcuna copia.
    </li>
    <li>
        <strong>Password:</strong> pone l'esportazione in un contenitore cifrato <strong>.dbx</strong>. Protegge il file durante il trasporto, non il database in sé — chi riceve la password può
        aprirlo — e l'origine resta leggibile anche senza di essa.
    </li>
</ul>
<p>
    La tua identità dell'emittente, la chiave che firma le tue marche, si crea automaticamente alla prima esportazione marcata con la sua origine. Consultala, esportala o rigenerala dalla scheda
    <strong>Identità dell'emittente</strong> delle impostazioni.
</p>
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
