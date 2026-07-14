// Contenuto della guida in italiano (tradotto da en.js).
// Ogni valore è l'HTML interno letterale della scheda corrispondente di HelpModal.
export default {
    manual: `
                    <h3>Introduzione</h3>
                    <p>
                        blunderDB è un software per creare database di posizioni di backgammon. Il suo punto di forza principale è offrire un unico luogo in cui raccogliere le posizioni che un giocatore ha incontrato (online,
                        nei tornei) e poter ristudiare queste posizioni filtrandole secondo vari filtri combinabili arbitrariamente. blunderDB può anche essere usato per creare cataloghi
                        di posizioni di riferimento.
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
                        <li>organizzare i match in tornei.</li>
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
                        <li>calcolare i valori di EPC per posizioni di bearoff (pannello EPC),</li>
                        <li>consultare i filtri di ricerca salvati (pannello Libreria filtri),</li>
                        <li>consultare la cronologia delle ricerche (pannello Cronologia ricerche),</li>
                        <li>vedere i log delle operazioni (pannello Log).</li>
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
                        Premere il tasto <strong>Tab</strong> apre il pannello di ricerca e permette di modificare una posizione sul tavoliere per aggiungerla al database o per definire una struttura di posizione da cercare.
                        La distribuzione delle pedine, il cube, il punteggio e il turno possono essere modificati con il mouse.
                    </p>

                    <h3>Riga di comando</h3>
                    <p>
                        La riga di comando, integrata nella barra di stato, permette di eseguire tutte le funzionalità di blunderDB: operazioni sul database, navigazione tra le posizioni, mostrare analisi e
                        commenti, cercare posizioni con filtri... Dopo aver preso confidenza con l'interfaccia, è consigliabile usare progressivamente la riga di comando, che permette un uso potente e
                        fluido di blunderDB, soprattutto per le funzionalità di ricerca delle posizioni.
                    </p>
                    <p>
                        Per aprire la riga di comando, premi il tasto <strong>Spazio</strong>. Appare un prompt nella barra di stato. Digita il tuo comando e premi <strong>Enter</strong> per eseguirlo. Premi
                        <strong>Escape</strong>
                        per annullare. La cronologia dei comandi e i risultati vengono registrati nel pannello <strong>Log</strong>.
                    </p>
                    <p>
                        blunderDB esegue le query inviate dall'utente purché siano valide e modifica immediatamente lo stato del database se necessario. Non sono richieste azioni di salvataggio esplicite
                        da parte dell'utente.
                    </p>
                    <p>
                        Per affinare una ricerca all'interno di posizioni già filtrate, usa il comando <strong>ss</strong> seguito da filtri (es. <strong>ss nc</strong>). Questo restringe la ricerca alle
                        sole posizioni attualmente mostrate, permettendo di restringere progressivamente i risultati. Il pannello di ricerca (<strong>Ctrl+F</strong>) offre anche una casella "Cerca nei risultati correnti"
                        per la stessa funzionalità.
                    </p>

                    <h3>Calcolatore EPC</h3>
                    <p>Il calcolatore EPC (Effective Pip Count) calcola il pip count effettivo delle posizioni di bearoff. Utilizza il database di bearoff a un lato a 6 punti di GNUbg per ottenere valori di EPC esatti.</p>
                    <p>
                        Per aprire il pannello EPC, premi <strong>Ctrl+E</strong>, fai clic sulla scheda EPC del pannello inferiore o digita <strong>epc</strong> nella riga di comando. Il tavoliere viene inizializzato con una configurazione standard
                        di bearoff (15 pedine).
                    </p>
                    <p>
                        Puoi aggiungere o rimuovere liberamente pedine sui punti del proprio quadrante interno con il mouse. I valori di EPC vengono mostrati in tempo reale nel pannello EPC dedicato, indicando per ciascun giocatore:
                    </p>
                    <ul>
                        <li><strong>EPC</strong>: il numero medio di pip necessari per portare a casa tutte le pedine,</li>
                        <li><strong>Pip Count</strong>: il pip count grezzo,</li>
                        <li><strong>Wastage</strong>: la differenza tra EPC e pip count,</li>
                        <li><strong>Avg Rolls</strong>: numero medio di lanci per portare a casa tutte le pedine,</li>
                        <li><strong>Std Dev</strong>: deviazione standard del numero di lanci.</li>
                    </ul>
                    <p>Quando entrambi i giocatori hanno pedine nel proprio quadrante interno, una sezione di confronto mostra le differenze di EPC e di pip count.</p>
                    <p>Per chiudere il pannello EPC, premi di nuovo <strong>Ctrl+E</strong> oppure passa a un'altra scheda.</p>

                    <h3>Navigazione dei match</h3>
                    <p>
                        blunderDB permette di scorrere le mosse dei match importati. Apri il pannello Match con <strong>Ctrl+Tab</strong> e fai doppio clic su un match (o premi <strong>Enter</strong>)
                        per caricarne le posizioni.
                    </p>
                    <p>
                        Durante la navigazione di un match, l'ultima posizione visitata viene salvata e ripristinata automaticamente. Usa i tasti <strong>Sinistra</strong>/<strong>Destra</strong> per spostarti tra le posizioni, e
                        <strong>PageUp</strong>/<strong>PageDown</strong> per saltare tra le partite.
                    </p>
                    <p>
                        Il pannello di analisi (<strong>Ctrl+L</strong>) mostra l'analisi di ogni mossa, evidenziando la mossa giocata. Premi <strong>d</strong> per alternare tra analisi delle pedine e del cube.
                    </p>

                    <h3>Collezioni</h3>
                    <p>
                        Le collezioni permettono di organizzare le posizioni in gruppi personalizzati. Apri il pannello Collezione con <strong>Ctrl+B</strong>, poi fai doppio clic su una collezione per scorrerne le posizioni.
                        Le collezioni e le posizioni che contengono possono essere riordinate trascinandole.
                    </p>

                    <h3>Anki (ripetizione dilazionata)</h3>
                    <p>Il pannello Anki (<strong>Ctrl+K</strong>) offre la ripetizione dilazionata per studiare posizioni di backgammon usando l'algoritmo FSRS.</p>
                    <p>
                        <strong>Creare mazzi:</strong> Fai clic su <em>Nuovo mazzo</em> per creare un mazzo a partire da una collezione o dai risultati di ricerca correnti. I mazzi basati su ricerche si sincronizzano automaticamente quando si attiva la scheda Anki.
                    </p>
                    <p>
                        <strong>Ripassare:</strong> Seleziona un mazzo e fai clic su <em>Studia</em> (o fai doppio clic su un mazzo) per iniziare a ripassare le carte in scadenza. Ogni carta mostra la posizione corrispondente sul
                        tavoliere. Valuta il tuo ricordo con i tasti <strong>1</strong> (Di nuovo), <strong>2</strong> (Difficile), <strong>3</strong> (Bene) o <strong>4</strong> (Facile). Premi <strong>Esc</strong> per fermarti
                        e tornare alla lista dei mazzi.
                    </p>
                    <p>
                        <strong>Ferma/Riprendi:</strong> Puoi fermare una sessione di ripasso in qualsiasi momento premendo <strong>Esc</strong>. Il pulsante cambia in <em>Riprendi</em> mostrando i tuoi progressi. Fai clic su di esso per
                        continuare da dove avevi lasciato.
                    </p>
                    <p>
                        <strong>Gestione dei mazzi:</strong> Usa i pulsanti di azione per rinominare, sincronizzare, reimpostare o eliminare i mazzi. I parametri di FSRS (ritenzione obiettivo, intervallo massimo, fuzz) possono essere configurati per ogni mazzo
                        nelle Impostazioni (icona a ingranaggio).
                    </p>

                    <h3>Tornei</h3>
                    <p>I tornei permettono di raggruppare i match per evento. Apri il pannello Torneo con <strong>Ctrl+Y</strong> per gestire i tornei e assegnare loro i match.</p>

                    <h3>Stats</h3>
                    <p>
                        Il pannello Stats (<strong>Ctrl+D</strong>) mostra statistiche di rendimento (PR e costo in MWC) calcolate a partire da tutte le posizioni importate. Usa la barra dei filtri per restringere l'analisi per
                        giocatore, torneo, intervallo di date, tipo di decisione o lunghezza del match. Fai clic su un qualsiasi indicatore per approfondire le posizioni corrispondenti.
                    </p>
`,
    shortcuts: `
                    <h3>Database</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Scorciatoia</th>
                                <th>Descrizione</th>
                            </tr>
                        </thead>

                        <tbody>
                            <tr>
                                <td>Ctrl + N</td>
                                <td>Nuovo database</td>
                            </tr>

                            <tr>
                                <td>Ctrl + O</td>
                                <td>Apri database</td>
                            </tr>

                            <tr>
                                <td>Ctrl + Shift + I</td>
                                <td>Importa database</td>
                            </tr>

                            <tr>
                                <td>Ctrl + Shift + S</td>
                                <td>Esporta database</td>
                            </tr>

                            <tr>
                                <td>Ctrl + Q</td>
                                <td>Esci da blunderDB</td>
                            </tr>

                            <tr>
                                <td>Ctrl + M</td>
                                <td>Modifica metadati</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Posizione</h3>

                    <table>
                        <thead>
                            <tr>
                                <th>Scorciatoia</th>
                                <th>Descrizione</th>
                            </tr>
                        </thead>

                        <tbody>
                            <tr>
                                <td>Ctrl + I</td>
                                <td>Importa posizione o match</td>
                            </tr>

                            <tr>
                                <td>Ctrl + C</td>
                                <td>Copia posizione (copia anche negli appunti del tavoliere)</td>
                            </tr>

                            <tr>
                                <td>Ctrl + X</td>
                                <td>Copia immagine del tavoliere negli appunti (PNG)</td>
                            </tr>

                            <tr>
                                <td>Ctrl + X, Ctrl + X</td>
                                <td>Copia immagine del tavoliere + analisi negli appunti (PNG)</td>
                            </tr>

                            <tr>
                                <td>Ctrl + V</td>
                                <td>Incolla posizione (nel pannello di ricerca: incolla sul tavoliere)</td>
                            </tr>

                            <tr>
                                <td>Ctrl + S</td>
                                <td>Salva posizione</td>
                            </tr>

                            <tr>
                                <td>Ctrl + U</td>
                                <td>Aggiorna posizione</td>
                            </tr>

                            <tr>
                                <td>Del</td>
                                <td>Elimina posizione</td>
                            </tr>

                            <tr>
                                <td>Backspace</td>
                                <td>Reimposta tavoliere, cube, punteggio e dadi</td>
                            </tr>

                            <tr>
                                <td>Ctrl + G</td>
                                <td>Mostra metadati della posizione</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Navigazione</h3>

                    <table>
                        <thead>
                            <tr>
                                <th>Scorciatoia</th>
                                <th>Descrizione</th>
                            </tr>
                        </thead>

                        <tbody>
                            <tr>
                                <td>Ctrl + R</td>
                                <td>Ricarica tutte le posizioni</td>
                            </tr>

                            <tr>
                                <td>PageUp, h</td>
                                <td>Prima posizione / Partita precedente (navigazione del match)</td>
                            </tr>

                            <tr>
                                <td>Left, k</td>
                                <td>Posizione precedente</td>
                            </tr>

                            <tr>
                                <td>Right, j</td>
                                <td>Posizione successiva</td>
                            </tr>

                            <tr>
                                <td>PageDown, l</td>
                                <td>Ultima posizione / Partita successiva (navigazione del match)</td>
                            </tr>

                            <tr>
                                <td>r</td>
                                <td>Carica posizione casuale</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Visualizzazione</h3>

                    <table>
                        <thead>
                            <tr>
                                <th>Scorciatoia</th>
                                <th>Descrizione</th>
                            </tr>
                        </thead>

                        <tbody>
                            <tr>
                                <td>Ctrl + ArrowLeft</td>
                                <td>Orienta il tavoliere a sinistra</td>
                            </tr>

                            <tr>
                                <td>Ctrl + ArrowRight</td>
                                <td>Orienta il tavoliere a destra</td>
                            </tr>

                            <tr>
                                <td>p</td>
                                <td>Mostra/nascondi pipcount</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Azioni</h3>

                    <table>
                        <thead>
                            <tr>
                                <th>Scorciatoia</th>
                                <th>Descrizione</th>
                            </tr>
                        </thead>

                        <tbody>
                            <tr>
                                <td>Tab</td>
                                <td>Apri pannello di ricerca (editor di posizione)</td>
                            </tr>

                            <tr>
                                <td>Space</td>
                                <td>Apri riga di comando</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Strumenti</h3>

                    <table>
                        <thead>
                            <tr>
                                <th>Scorciatoia</th>
                                <th>Descrizione</th>
                            </tr>
                        </thead>

                        <tbody>
                            <tr>
                                <td>Ctrl + L</td>
                                <td>Mostra analisi</td>
                            </tr>

                            <tr>
                                <td>Ctrl + P</td>
                                <td>Scrivi commenti</td>
                            </tr>

                            <tr>
                                <td>Ctrl + K</td>
                                <td>Mostra pannello Anki</td>
                            </tr>

                            <tr>
                                <td>Ctrl + F</td>
                                <td>Pannello di ricerca</td>
                            </tr>

                            <tr>
                                <td>Ctrl + Tab</td>
                                <td>Pannello Match</td>
                            </tr>

                            <tr>
                                <td>Ctrl + B</td>
                                <td>Pannello Collezione</td>
                            </tr>

                            <tr>
                                <td>Ctrl + Y</td>
                                <td>Pannello Tornei</td>
                            </tr>

                            <tr>
                                <td>Ctrl + D</td>
                                <td>Pannello Stats</td>
                            </tr>

                            <tr>
                                <td>Ctrl + E</td>
                                <td>Pannello EPC</td>
                            </tr>

                            <tr>
                                <td>?</td>
                                <td>Apri guida</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Riga di comando</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Scorciatoia</th>
                                <th>Descrizione</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>Up</td>
                                <td>Scorri la cronologia dei comandi verso l'alto</td>
                            </tr>
                            <tr>
                                <td>Down</td>
                                <td>Scorri la cronologia dei comandi verso il basso</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Pannello di analisi</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Scorciatoia</th>
                                <th>Descrizione</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>Click</td>
                                <td>Seleziona/deseleziona una mossa (mostra/nascondi frecce)</td>
                            </tr>
                            <tr>
                                <td>Up, k</td>
                                <td>Seleziona la mossa precedente (quando una mossa è selezionata)</td>
                            </tr>
                            <tr>
                                <td>Down, j</td>
                                <td>Seleziona la mossa successiva (quando una mossa è selezionata)</td>
                            </tr>
                            <tr>
                                <td>d</td>
                                <td>Alterna tra analisi delle pedine e del cube (solo nella navigazione del match)</td>
                            </tr>
                            <tr>
                                <td>Esc</td>
                                <td>Deseleziona la mossa. Se nessuna mossa è selezionata, chiudi il pannello.</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Pannello cronologia ricerche</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Scorciatoia</th>
                                <th>Descrizione</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>Click</td>
                                <td>Seleziona/deseleziona una ricerca (mostra posizione)</td>
                            </tr>
                            <tr>
                                <td>Double-click</td>
                                <td>Esegui la ricerca e chiudi il pannello</td>
                            </tr>
                            <tr>
                                <td>Up, k</td>
                                <td>Seleziona la ricerca precedente (più recente, sopra)</td>
                            </tr>
                            <tr>
                                <td>Down, j</td>
                                <td>Seleziona la ricerca successiva (più vecchia, sotto)</td>
                            </tr>
                            <tr>
                                <td>Esc</td>
                                <td>Deseleziona la ricerca. Se nessuna ricerca è selezionata, chiudi il pannello.</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Pannello Libreria filtri</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Scorciatoia</th>
                                <th>Descrizione</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>Click</td>
                                <td>Seleziona/deseleziona un filtro (mostra posizione)</td>
                            </tr>
                            <tr>
                                <td>Double-click</td>
                                <td>Esegui la ricerca del filtro</td>
                            </tr>
                            <tr>
                                <td>Up, k</td>
                                <td>Seleziona il filtro precedente (quando un filtro è selezionato)</td>
                            </tr>
                            <tr>
                                <td>Down, j</td>
                                <td>Seleziona il filtro successivo (quando un filtro è selezionato)</td>
                            </tr>
                            <tr>
                                <td>Esc</td>
                                <td>Deseleziona il filtro. Se nessun filtro è selezionato, chiudi il pannello.</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Pannello Match</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Scorciatoia</th>
                                <th>Descrizione</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>Click</td>
                                <td>Seleziona/deseleziona un match</td>
                            </tr>
                            <tr>
                                <td>Double-click</td>
                                <td>Carica le posizioni del match</td>
                            </tr>
                            <tr>
                                <td>Up, k</td>
                                <td>Seleziona il match precedente</td>
                            </tr>
                            <tr>
                                <td>Down, j</td>
                                <td>Seleziona il match successivo</td>
                            </tr>
                            <tr>
                                <td>Del</td>
                                <td>Elimina il match selezionato</td>
                            </tr>
                            <tr>
                                <td>Esc</td>
                                <td>Deseleziona il match. Se nessun match è selezionato, chiudi il pannello.</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Pannello Collezione</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Scorciatoia</th>
                                <th>Descrizione</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>Click</td>
                                <td>Seleziona una collezione (mostra le sue posizioni)</td>
                            </tr>
                            <tr>
                                <td>Double-click</td>
                                <td>Apri la collezione e scorri le sue posizioni</td>
                            </tr>
                            <tr>
                                <td>Del</td>
                                <td>Rimuovi la posizione corrente dalla collezione attiva</td>
                            </tr>
                            <tr>
                                <td>Esc</td>
                                <td>Deseleziona la collezione. Se nessuna collezione è selezionata, chiudi il pannello.</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Pannello Anki</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Scorciatoia</th>
                                <th>Descrizione</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>1</td>
                                <td>Valuta: Di nuovo (ripasso fallito, mostra di nuovo presto)</td>
                            </tr>
                            <tr>
                                <td>2</td>
                                <td>Valuta: Difficile (ricordo difficile)</td>
                            </tr>
                            <tr>
                                <td>3</td>
                                <td>Valuta: Bene (ricordo corretto)</td>
                            </tr>
                            <tr>
                                <td>4</td>
                                <td>Valuta: Facile (ricordo senza sforzo)</td>
                            </tr>
                            <tr>
                                <td>Esc</td>
                                <td>Ferma il ripasso e torna alla lista dei mazzi (si può riprendere in seguito)</td>
                            </tr>
                        </tbody>
                    </table>
`,
    commands: `
                    <h3>Database</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Comando</th>
                                <th>Descrizione</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>new, ne, n</td>
                                <td>Crea un nuovo database</td>
                            </tr>
                            <tr>
                                <td>open, op, o</td>
                                <td>Apri un database esistente</td>
                            </tr>
                            <tr>
                                <td>import_db, idb</td>
                                <td>Importa e unisci un altro database</td>
                            </tr>
                            <tr>
                                <td>export_db, edb</td>
                                <td>Esporta la selezione corrente in un nuovo database</td>
                            </tr>
                            <tr>
                                <td>quit, q</td>
                                <td>Esci da blunderDB</td>
                            </tr>
                            <tr>
                                <td>meta</td>
                                <td>Modifica metadati</td>
                            </tr>
                        </tbody>
                    </table>
                    <h3>Posizione</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Comando</th>
                                <th>Descrizione</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>import, i</td>
                                <td>Importa una posizione o un match</td>
                            </tr>
                            <tr>
                                <td>write, wr, w</td>
                                <td>Salva una posizione</td>
                            </tr>
                            <tr>
                                <td>write!, wr!, w!</td>
                                <td>Aggiorna una posizione</td>
                            </tr>
                            <tr>
                                <td>delete, del, d</td>
                                <td>Elimina una posizione</td>
                            </tr>
                        </tbody>
                    </table>
                    <h3>Navigazione</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Comando</th>
                                <th>Descrizione</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>[numero]</td>
                                <td>Vai a una posizione specifica per indice</td>
                            </tr>
                        </tbody>
                    </table>
                    <h3>Strumenti</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Comando</th>
                                <th>Descrizione</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>list, l</td>
                                <td>Mostra analisi</td>
                            </tr>
                            <tr>
                                <td>comment, co</td>
                                <td>Scrivi commenti</td>
                            </tr>
                            <tr>
                                <td>filter, fl</td>
                                <td>Mostra la libreria dei filtri</td>
                            </tr>
                            <tr>
                                <td>history, hi</td>
                                <td>Mostra la cronologia delle ricerche</td>
                            </tr>
                            <tr>
                                <td>match, ma</td>
                                <td>Mostra il pannello Match</td>
                            </tr>
                            <tr>
                                <td>collection, coll</td>
                                <td>Mostra il pannello delle collezioni</td>
                            </tr>
                            <tr>
                                <td>epc</td>
                                <td>Calcolatore EPC (Effective Pip Count)</td>
                            </tr>
                            <tr>
                                <td>m</td>
                                <td>Naviga l'ultimo match visitato</td>
                            </tr>
                            <tr>
                                <td>help, he, h</td>
                                <td>Apri guida</td>
                            </tr>
                            <tr>
                                <td>met</td>
                                <td>Apri la tabella di match equity Kazaross-XG2</td>
                            </tr>
                            <tr>
                                <td>tp2</td>
                                <td>Take point con cube a 2 (Live e Last)</td>
                            </tr>
                            <tr>
                                <td>tp2_live</td>
                                <td>Take point con cube a 2 nelle corse lunghe</td>
                            </tr>
                            <tr>
                                <td>tp2_last</td>
                                <td>Take point con cube a 2 nelle posizioni di ultimo lancio</td>
                            </tr>
                            <tr>
                                <td>tp4</td>
                                <td>Take point con cube a 4 (Live e Last)</td>
                            </tr>
                            <tr>
                                <td>tp4_live</td>
                                <td>Take point con cube a 4 nelle corse lunghe</td>
                            </tr>
                            <tr>
                                <td>tp4_last</td>
                                <td>Take point con cube a 4 nelle posizioni di ultimo lancio</td>
                            </tr>
                            <tr>
                                <td>gv1</td>
                                <td>Valori di gammon con cube a 1</td>
                            </tr>
                            <tr>
                                <td>gv2</td>
                                <td>Valori di gammon con cube a 2</td>
                            </tr>
                            <tr>
                                <td>gv4</td>
                                <td>Valori di gammon con cube a 4</td>
                            </tr>
                            <tr>
                                <td>#tag1 tag2 ...</td>
                                <td>Etichetta posizione</td>
                            </tr>
                            <tr>
                                <td>s</td>
                                <td>Cerca posizioni con filtri</td>
                            </tr>
                            <tr>
                                <td>ss</td>
                                <td>Cerca nei risultati correnti con filtri</td>
                            </tr>
                            <tr>
                                <td>e</td>
                                <td>Ricarica tutte le posizioni</td>
                            </tr>
                        </tbody>
                    </table>
                    <h3>Filtri</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Filtro</th>
                                <th>Descrizione</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>cube, cub, cu, c</td>
                                <td>Includi il cube</td>
                            </tr>
                            <tr>
                                <td>score, sco, sc, s</td>
                                <td>Includi il punteggio</td>
                            </tr>
                            <tr>
                                <td>d</td>
                                <td>Includi il tipo di decisione</td>
                            </tr>
                            <tr>
                                <td>D</td>
                                <td>Includi la tirata di dadi</td>
                            </tr>
                            <tr>
                                <td>nc</td>
                                <td>Includi posizioni senza contatto</td>
                            </tr>
                            <tr>
                                <td>M</td>
                                <td>Includi posizioni speculari</td>
                            </tr>
                            <tr>
                                <td>i</td>
                                <td>Solo posizioni importate singolarmente, non portate da un import di partita</td>
                            </tr>
                            <tr>
                                <td>x</td>
                                <td
                                    >Escludi le posizioni che contengono <em>una qualsiasi</em> pedina della struttura disegnata nella scheda "Except" (es. disegnare pedine su 1, 3 e 5 conserva solo le posizioni che non ne hanno nessuna
                                    di esse). Alterna "At least" / "Except" sopra i filtri per disegnare le pedine escluse sul tavoliere (mostrate con un segnale rosso). Il conteggio per punto non è limitato (3 su un punto
                                    esclude 3 o più lì — un punto fatto senza pedina di riserva), e due clic rapidi su un punto lo segnano come obbligatoriamente vuoto (una cella con tratteggio rosso, di qualsiasi colore); un solo clic su quel
                                    punto lo sblocca. Su un punto condiviso, "Except" prevale su "At least" quando i due si contraddicono.</td
                                >
                            </tr>
                            <tr>
                                <td>p>x</td>
                                <td>Pip count &gt; x</td>
                            </tr>
                            <tr>
                                <td>p&lt;x</td>
                                <td>Pip count &lt; x</td>
                            </tr>
                            <tr>
                                <td>px,y</td>
                                <td>Pip count tra x e y</td>
                            </tr>
                            <tr>
                                <td>P>x</td>
                                <td>Pip count assoluto del giocatore &gt; x</td>
                            </tr>
                            <tr>
                                <td>P&lt;x</td>
                                <td>Pip count assoluto del giocatore &lt; x</td>
                            </tr>
                            <tr>
                                <td>Px,y</td>
                                <td>Pip count assoluto del giocatore tra x e y</td>
                            </tr>
                            <tr>
                                <td>e>x</td>
                                <td>Equity &gt; x (in millipunti)</td>
                            </tr>
                            <tr>
                                <td>e&lt;x</td>
                                <td>Equity &lt; x (in millipunti)</td>
                            </tr>
                            <tr>
                                <td>ex,y</td>
                                <td>Equity tra x e y (in millipunti)</td>
                            </tr>
                            <tr>
                                <td>E>x</td>
                                <td>Errore di mossa del giocatore 1 &gt; x (in millipunti)</td>
                            </tr>
                            <tr>
                                <td>E&lt;x</td>
                                <td>Errore di mossa del giocatore 1 &lt; x (in millipunti)</td>
                            </tr>
                            <tr>
                                <td>Ex,y</td>
                                <td>Errore di mossa del giocatore 1 tra x e y (in millipunti)</td>
                            </tr>
                            <tr>
                                <td>w>x</td>
                                <td>Tasso di vittoria &gt; x</td>
                            </tr>
                            <tr>
                                <td>w&lt;x</td>
                                <td>Tasso di vittoria &lt; x</td>
                            </tr>
                            <tr>
                                <td>wx,y</td>
                                <td>Tasso di vittoria tra x e y</td>
                            </tr>
                            <tr>
                                <td>g>x</td>
                                <td>Tasso di gammon &gt; x</td>
                            </tr>
                            <tr>
                                <td>g&lt;x</td>
                                <td>Tasso di gammon &lt; x</td>
                            </tr>
                            <tr>
                                <td>gx,y</td>
                                <td>Tasso di gammon tra x e y</td>
                            </tr>
                            <tr>
                                <td>b>x</td>
                                <td>Tasso di backgammon &gt; x</td>
                            </tr>
                            <tr>
                                <td>b&lt;x</td>
                                <td>Tasso di backgammon &lt; x</td>
                            </tr>
                            <tr>
                                <td>bx,y</td>
                                <td>Tasso di backgammon tra x e y</td>
                            </tr>
                            <tr>
                                <td>W>x</td>
                                <td>Tasso di vittoria dell'avversario &gt; x</td>
                            </tr>
                            <tr>
                                <td>W&lt;x</td>
                                <td>Tasso di vittoria dell'avversario &lt; x</td>
                            </tr>
                            <tr>
                                <td>Wx,y</td>
                                <td>Tasso di vittoria dell'avversario tra x e y</td>
                            </tr>
                            <tr>
                                <td>G>x</td>
                                <td>Tasso di gammon dell'avversario &gt; x</td>
                            </tr>
                            <tr>
                                <td>G&lt;x</td>
                                <td>Tasso di gammon dell'avversario &lt; x</td>
                            </tr>
                            <tr>
                                <td>Gx,y</td>
                                <td>Tasso di gammon dell'avversario tra x e y</td>
                            </tr>
                            <tr>
                                <td>B>x</td>
                                <td>Tasso di backgammon dell'avversario &gt; x</td>
                            </tr>
                            <tr>
                                <td>B&lt;x</td>
                                <td>Tasso di backgammon dell'avversario &lt; x</td>
                            </tr>
                            <tr>
                                <td>Bx,y</td>
                                <td>Tasso di backgammon dell'avversario tra x e y</td>
                            </tr>
                            <tr>
                                <td>o>x</td>
                                <td>Pedine portate a casa dal giocatore &gt; x</td>
                            </tr>
                            <tr>
                                <td>o&lt;x</td>
                                <td>Pedine portate a casa dal giocatore &lt; x</td>
                            </tr>
                            <tr>
                                <td>ox,y</td>
                                <td>Pedine portate a casa dal giocatore tra x e y</td>
                            </tr>
                            <tr>
                                <td>O>x</td>
                                <td>Pedine portate a casa dall'avversario &gt; x</td>
                            </tr>
                            <tr>
                                <td>O&lt;x</td>
                                <td>Pedine portate a casa dall'avversario &lt; x</td>
                            </tr>
                            <tr>
                                <td>Ox,y</td>
                                <td>Pedine portate a casa dall'avversario tra x e y</td>
                            </tr>
                            <tr>
                                <td>k>x</td>
                                <td>Pedine arretrate del giocatore &gt; x</td>
                            </tr>
                            <tr>
                                <td>k&lt;x</td>
                                <td>Pedine arretrate del giocatore &lt; x</td>
                            </tr>
                            <tr>
                                <td>kx,y</td>
                                <td>Pedine arretrate del giocatore tra x e y</td>
                            </tr>
                            <tr>
                                <td>K>x</td>
                                <td>Pedine arretrate dell'avversario &gt; x</td>
                            </tr>
                            <tr>
                                <td>K&lt;x</td>
                                <td>Pedine arretrate dell'avversario &lt; x</td>
                            </tr>
                            <tr>
                                <td>Kx,y</td>
                                <td>Pedine arretrate dell'avversario tra x e y</td>
                            </tr>
                            <tr>
                                <td>z>x</td>
                                <td>Pedine del giocatore nella zona &gt; x</td>
                            </tr>
                            <tr>
                                <td>z&lt;x</td>
                                <td>Pedine del giocatore nella zona &lt; x</td>
                            </tr>
                            <tr>
                                <td>zx,y</td>
                                <td>Pedine del giocatore nella zona tra x e y</td>
                            </tr>
                            <tr>
                                <td>Z>x</td>
                                <td>Pedine dell'avversario nella zona &gt; x</td>
                            </tr>
                            <tr>
                                <td>Z&lt;x</td>
                                <td>Pedine dell'avversario nella zona &lt; x</td>
                            </tr>
                            <tr>
                                <td>Zx,y</td>
                                <td>Pedine dell'avversario nella zona tra x e y</td>
                            </tr>
                            <tr>
                                <td>bo>x</td>
                                <td>Blot del giocatore nell'outfield &gt; x</td>
                            </tr>
                            <tr>
                                <td>bo&lt;x</td>
                                <td>Blot del giocatore nell'outfield &lt; x</td>
                            </tr>
                            <tr>
                                <td>box,y</td>
                                <td>Blot del giocatore nell'outfield tra x e y</td>
                            </tr>
                            <tr>
                                <td>BO>x</td>
                                <td>Blot dell'avversario nell'outfield &gt; x</td>
                            </tr>
                            <tr>
                                <td>BO&lt;x</td>
                                <td>Blot dell'avversario nell'outfield &lt; x</td>
                            </tr>
                            <tr>
                                <td>BOx,y</td>
                                <td>Blot dell'avversario nell'outfield tra x e y</td>
                            </tr>
                            <tr>
                                <td>bj&gt;x</td>
                                <td>Blot Jan del giocatore &gt; x</td>
                            </tr>
                            <tr>
                                <td>bj&lt;x</td>
                                <td>Blot Jan del giocatore &lt; x</td>
                            </tr>
                            <tr>
                                <td>bjx,y</td>
                                <td>Blot Jan del giocatore tra x e y</td>
                            </tr>
                            <tr>
                                <td>BJ&gt;x</td>
                                <td>Blot Jan dell'avversario &gt; x</td>
                            </tr>
                            <tr>
                                <td>BJ&lt;x</td>
                                <td>Blot Jan dell'avversario &lt; x</td>
                            </tr>
                            <tr>
                                <td>BJx,y</td>
                                <td>Blot Jan dell'avversario tra x e y</td>
                            </tr>

                            <tr>
                                <td>t"word1;word2;..."</td>
                                <td>Cerca testo</td>
                            </tr>
                            <tr>
                                <td>m"pattern1;pattern2;..."</td>
                                <td>Mosse migliori che contengono almeno uno dei pattern indicati</td>
                            </tr>
                            <tr>
                                <td>m"ND;DT;DP;..."</td>
                                <td>Migliore decisione sul cube tra No Double/Take, Double/Take, Double/Pass</td>
                            </tr>
                            <tr>
                                <td>T&gt;x</td>
                                <td>Data di creazione &gt; x (anno/mese/giorno)</td>
                            </tr>
                            <tr>
                                <td>T&lt;x</td>
                                <td>Data di creazione &lt; x (anno/mese/giorno)</td>
                            </tr>
                            <tr>
                                <td>Tx,y</td>
                                <td>Data di creazione tra x e y</td>
                            </tr>
                            <tr>
                                <td>max</td>
                                <td>Cerca nel match con ID x (es. ma3)</td>
                            </tr>
                            <tr>
                                <td>max,y</td>
                                <td>Cerca nei match con ID da x a y (es. ma2,5)</td>
                            </tr>
                            <tr>
                                <td>tnx</td>
                                <td>Cerca nel torneo con ID x (es. tn1)</td>
                            </tr>
                            <tr>
                                <td>tnx,y</td>
                                <td>Cerca nei tornei con ID da x a y (es. tn1,3)</td>
                            </tr>
                        </tbody>
                    </table>
                    <h3>Varie</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Comando</th>
                                <th>Descrizione</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>clear, cl</td>
                                <td>Cancella la cronologia dei comandi</td>
                            </tr>
                            <tr>
                                <td>migrate_from_1_0_to_1_1</td>
                                <td>Migra il database dalla versione 1.0 alla 1.1</td>
                            </tr>
                            <tr>
                                <td>migrate_from_1_1_to_1_2</td>
                                <td>Migra il database dalla versione 1.1 alla 1.2</td>
                            </tr>
                            <tr>
                                <td>migrate_from_1_2_to_1_3</td>
                                <td>Migra il database dalla versione 1.2 alla 1.3</td>
                            </tr>
                        </tbody>
                    </table>
`,
    about: `
                    <h3>Versione</h3>
                    <p>Versione dell'applicazione: {appVersion}</p>
                    <p>Versione del database: {dbVersion}</p>

                    <h3>Autore</h3>
                    <p><strong>Kévin Unger &lt;blunderdb@proton.me&gt;</strong></p>
                    <p>Puoi anche trovarmi su Heroes con il nickname <strong>postmanpat</strong>.</p>
                    <p>
                        Ho sviluppato blunderDB inizialmente per uso personale, per individuare schemi nei miei errori. Ma è molto piacevole ricevere riscontri, soprattutto quando si sono dedicate molte ore
                        alla progettazione, alla programmazione, al debug... Quindi non esitare a scrivermi per condividere i tuoi riscontri.
                    </p>
                    <p>Ecco diversi modi per contattarmi:</p>
                    <ul>
                        <li>Parla con me se ci incontriamo a un torneo,</li>
                        <li>Inviami un'email,</li>
                    </ul>
                    <h3>Licenza</h3>
                    <p>
                        blunderDB è distribuito sotto la licenza MIT. Questo significa che sei libero di usare, copiare, modificare, unire, pubblicare, distribuire, sublicenziare e/o vendere copie del software, a condizione
                        che l'avviso di copyright originale e questo avviso di permesso siano inclusi in tutte le copie o porzioni sostanziali del software.
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
                    <p>La tabella di match equity Kazaross-XG2 (MET) è attribuita a <strong>Neil Kazaross</strong>.</p>
                    <p>Le tabelle dei take point e dei valori di gammon sono tratte dal libro <em>The Theory of Backgammon</em> di <strong>Dirk Schiemann</strong>.</p>
                    <p>
                        Il database di bearoff a un lato a 6 punti utilizzato per il calcolo dell'EPC (Effective Pip Count) è stato generato con <strong>GNU Backgammon</strong> (GNUbg). GNUbg è un software libero e open source
                        di backgammon distribuito sotto la GNU General Public License.
                    </p>
`
};
