// Deutscher Hilfe-Inhalt (Übersetzung von en.js).
// Jeder Wert ist das innere HTML des entsprechenden HelpModal-Tabs.
export default {
    manual: `
                    <h3>Einführung</h3>
                    <p>
                        blunderDB ist eine Software zum Erstellen von Backgammon-Positionsdatenbanken. Ihre größte Stärke besteht darin, einen einzigen Ort zu bieten, an dem ein Spieler die von ihm erlebten Positionen (online,
                        in Turnieren) zusammenführen und diese Positionen erneut studieren kann, indem er sie nach verschiedenen, beliebig kombinierbaren Filtern filtert. blunderDB kann auch verwendet werden, um Kataloge
                        von Referenzpositionen zu erstellen.
                    </p>
                    <p>Positionen werden in einer Datenbank gespeichert, die durch eine .db-Datei dargestellt wird.</p>

                    <h3>Hauptinteraktionen</h3>
                    <p>Die wichtigsten möglichen Interaktionen mit blunderDB sind:</p>
                    <ul>
                        <li>eine neue Position hinzufügen,</li>
                        <li>eine bestehende Position ändern,</li>
                        <li>das Brett als PNG-Bild in die Zwischenablage kopieren (<strong>Ctrl+X</strong>) oder das Brett mit seiner Analyse (<strong>Ctrl+X, Ctrl+X</strong>),</li>
                        <li>eine bestehende Position löschen,</li>
                        <li>nach einer oder mehreren Positionen suchen,</li>
                        <li>Matches aus verschiedenen Quellen importieren (XG, GnuBG, BGBlitz, Jellyfish), einschließlich Kommentaren aus XG-Dateien,</li>
                        <li>die Züge eines importierten Matches durchsehen,</li>
                        <li>Positionen in Sammlungen organisieren,</li>
                        <li>Matches in Turnieren organisieren.</li>
                    </ul>
                    <p>Der Benutzer kann Positionen frei taggen und mit Kommentaren versehen.</p>

                    <h3>Beschreibung der Benutzeroberfläche</h3>
                    <p>Die Benutzeroberfläche von blunderDB ist von oben nach unten wie folgt aufgebaut:</p>
                    <ul>
                        <li>[oben] die Symbolleiste, die alle wichtigen Operationen zusammenfasst, die an der Datenbank durchgeführt werden können,</li>
                        <li>[in der Mitte] der Hauptanzeigebereich, der das Anzeigen oder Bearbeiten von Backgammon-Positionen ermöglicht,</li>
                        <li>[unten] die Statusleiste, die die Befehlszeile integriert und verschiedene Informationen über die aktuelle Position anzeigt.</li>
                    </ul>
                    <p>Panels können angezeigt werden, um:</p>
                    <ul>
                        <li>die mit der aktuellen Position verknüpften Analysedaten anzuzeigen (aus XG, GnuBG oder BGBlitz),</li>
                        <li>Kommentare anzuzeigen, hinzuzufügen oder zu ändern,</li>
                        <li>importierte Matches durchzusehen und durch ihre Züge zu navigieren (Match-Panel),</li>
                        <li>Sammlungen von Positionen zu verwalten (Collection-Panel),</li>
                        <li>Positionen mit verteilter Wiederholung zu studieren (Anki-Panel),</li>
                        <li>Turniere zu verwalten (Tournament-Panel),</li>
                        <li>Leistungsstatistiken anzuzeigen (Stats-Panel),</li>
                        <li>EPC-Werte für Auswürfelpositionen zu berechnen (EPC-Panel),</li>
                        <li>gespeicherte Suchfilter durchzusehen (Filter Library-Panel),</li>
                        <li>den Suchverlauf durchzusehen (Search History-Panel),</li>
                        <li>Betriebsprotokolle anzuzeigen (Log-Panel).</li>
                    </ul>
                    <p>Der Hauptanzeigebereich bietet dem Benutzer:</p>
                    <ul>
                        <li>ein Brett zum Anzeigen oder Bearbeiten einer Backgammon-Position,</li>
                        <li>die Höhe und den Besitzer des Cubes,</li>
                        <li>den Pip-Count jedes Spielers,</li>
                        <li>den Spielstand jedes Spielers,</li>
                        <li>die zu spielenden Würfel. Wird kein Wert auf den Würfeln angezeigt, gibt die Position der Würfel an, welcher Spieler am Zug ist und dass die Position eine Cube-Entscheidung ist.</li>
                    </ul>
                    <p>Die Statusleiste zeigt von links nach rechts an:</p>
                    <ul>
                        <li>die Befehlszeile (zum Öffnen <strong>Space</strong> drücken),</li>
                        <li>eine Informationsmeldung zur zuletzt durchgeführten Operation,</li>
                        <li>den Index der aktuellen Position, gefolgt von der Gesamtzahl der Positionen (oder Zug-/Partie-Informationen bei der Navigation in einem Match).</li>
                    </ul>
                    <p>Bei Positionen, die aus einer Benutzersuche stammen, entspricht die in der Statusleiste angegebene Positionsanzahl der Anzahl der gefilterten Positionen.</p>

                    <h3>Positionen durchsehen</h3>
                    <p>Standardmäßig ermöglicht blunderDB Folgendes:</p>
                    <ul>
                        <li>durch die verschiedenen Positionen der aktuellen Bibliothek zu blättern,</li>
                        <li>die mit einer Position verknüpften Analyseinformationen anzuzeigen,</li>
                        <li>Kommentare zu einer Position anzuzeigen, hinzuzufügen und zu ändern.</li>
                    </ul>

                    <h3>Positionen bearbeiten</h3>
                    <p>
                        Durch Drücken der <strong>Tab</strong>-Taste wird das Suchpanel geöffnet, und es kann eine Position auf dem Brett bearbeitet werden, um sie der Datenbank hinzuzufügen oder eine zu suchende Positionsstruktur zu definieren.
                        Die Verteilung der Steine, der Cube, der Spielstand und das Zugrecht können mit der Maus geändert werden.
                    </p>

                    <h3>Befehlszeile</h3>
                    <p>
                        Die in die Statusleiste integrierte Befehlszeile ermöglicht alle Funktionen von blunderDB: Datenbankoperationen, Positionsnavigation, Anzeigen von Analysen und
                        Kommentaren, Suche nach Positionen mit Filtern... Sobald die Oberfläche vertraut ist, wird empfohlen, schrittweise die Befehlszeile zu verwenden, die eine leistungsstarke und
                        flüssige Nutzung von blunderDB ermöglicht, insbesondere für die Positionssuchfunktionen.
                    </p>
                    <p>
                        Zum Öffnen der Befehlszeile die <strong>Space</strong>-Taste drücken. In der Statusleiste erscheint eine Eingabeaufforderung. Befehl eingeben und zur Ausführung <strong>Enter</strong> drücken.
                        <strong>Escape</strong>
                        drücken, um abzubrechen. Befehlsverlauf und Ergebnisse werden im <strong>Log</strong>-Panel protokolliert.
                    </p>
                    <p>
                        blunderDB führt die vom Benutzer gesendeten Abfragen aus, sofern sie gültig sind, und ändert bei Bedarf sofort den Zustand der Datenbank. Es sind keine expliziten Speicheraktionen
                        durch den Benutzer erforderlich.
                    </p>
                    <p>
                        Um eine Suche innerhalb zuvor gefilterter Positionen zu verfeinern, den Befehl <strong>ss</strong> gefolgt von Filtern verwenden (z. B. <strong>ss nc</strong>). Dies schränkt die Suche auf
                        nur die aktuell angezeigten Positionen ein und ermöglicht eine schrittweise Eingrenzung der Ergebnisse. Das Suchpanel (<strong>Ctrl+F</strong>) bietet ebenfalls ein Kontrollkästchen „In aktuellen Ergebnissen suchen"
                        für dieselbe Funktion.
                    </p>

                    <h3>EPC-Rechner</h3>
                    <p>Der EPC-Rechner (Effective Pip Count) berechnet den effektiven Pip-Count von Auswürfelpositionen. Er verwendet die einseitige 6-Punkte-Auswürfeldatenbank von GnuBG für exakte EPC-Werte.</p>
                    <p>
                        Zum Öffnen des EPC-Panels <strong>Ctrl+E</strong> drücken, im unteren Panel auf den EPC-Tab klicken oder <strong>epc</strong> in die Befehlszeile eingeben. Das Brett wird mit einer Standard-Auswürfelkonfiguration
                        (15 Steine) initialisiert.
                    </p>
                    <p>
                        Sie können Steine auf den Feldern des Heimbretts mit der Maus frei hinzufügen oder entfernen. Die EPC-Werte werden in Echtzeit im dafür vorgesehenen EPC-Panel angezeigt und zeigen für jeden Spieler:
                    </p>
                    <ul>
                        <li><strong>EPC</strong>: die durchschnittliche Anzahl der Pips, die zum Auswürfeln aller Steine benötigt werden,</li>
                        <li><strong>Pip Count</strong>: der reine Pip-Count,</li>
                        <li><strong>Wastage</strong>: die Differenz zwischen EPC und Pip-Count,</li>
                        <li><strong>Avg Rolls</strong>: durchschnittliche Anzahl der Würfe zum Auswürfeln,</li>
                        <li><strong>Std Dev</strong>: Standardabweichung der Anzahl der Würfe.</li>
                    </ul>
                    <p>Wenn beide Spieler Steine in ihrem Heimbrett haben, zeigt ein Vergleichsbereich die EPC- und Pip-Count-Differenzen.</p>
                    <p>Zum Schließen des EPC-Panels erneut <strong>Ctrl+E</strong> drücken oder zu einem anderen Tab wechseln.</p>

                    <h3>Match-Navigation</h3>
                    <p>
                        blunderDB ermöglicht das Durchsehen der Züge importierter Matches. Das Match-Panel mit <strong>Ctrl+Tab</strong> öffnen und auf ein Match doppelklicken (oder <strong>Enter</strong> drücken),
                        um dessen Positionen zu laden.
                    </p>
                    <p>
                        Bei der Navigation in einem Match wird die zuletzt besuchte Position automatisch gespeichert und wiederhergestellt. Mit den Tasten <strong>Left</strong>/<strong>Right</strong> zwischen Positionen wechseln und mit
                        <strong>PageUp</strong>/<strong>PageDown</strong> zwischen Partien springen.
                    </p>
                    <p>
                        Das Analyse-Panel (<strong>Ctrl+L</strong>) zeigt die Analyse für jeden Zug, wobei der gespielte Zug hervorgehoben wird. <strong>d</strong> drücken, um zwischen Stein- und Cube-Analyse umzuschalten.
                    </p>

                    <h3>Sammlungen</h3>
                    <p>
                        Sammlungen ermöglichen das Organisieren von Positionen in benutzerdefinierten Gruppen. Das Collection-Panel mit <strong>Ctrl+B</strong> öffnen und dann auf eine Sammlung doppelklicken, um deren Positionen durchzusehen.
                        Sammlungen und die darin enthaltenen Positionen können per Drag-and-drop umgeordnet werden.
                    </p>

                    <h3>Anki (verteilte Wiederholung)</h3>
                    <p>Das Anki-Panel (<strong>Ctrl+K</strong>) bietet verteilte Wiederholung zum Studieren von Backgammon-Positionen mit dem FSRS-Algorithmus.</p>
                    <p>
                        <strong>Decks erstellen:</strong> Auf <em>New Deck</em> klicken, um ein Deck aus einer Sammlung oder aus den aktuellen Suchergebnissen zu erstellen. Suchbasierte Decks werden automatisch synchronisiert, wenn der Anki-Tab
                        aktiviert wird.
                    </p>
                    <p>
                        <strong>Wiederholen:</strong> Ein Deck auswählen und auf <em>Study</em> klicken (oder auf ein Deck doppelklicken), um mit dem Wiederholen fälliger Karten zu beginnen. Jede Karte zeigt die entsprechende Position auf dem
                        Brett. Bewerten Sie Ihr Erinnerungsvermögen mit den Tasten <strong>1</strong> (Again), <strong>2</strong> (Hard), <strong>3</strong> (Good) oder <strong>4</strong> (Easy). <strong>Esc</strong> drücken, um zu stoppen
                        und zur Deck-Liste zurückzukehren.
                    </p>
                    <p>
                        <strong>Stoppen/Fortsetzen:</strong> Sie können eine Wiederholungssitzung jederzeit durch Drücken von <strong>Esc</strong> stoppen. Die Schaltfläche wechselt zu <em>Resume</em> und zeigt Ihren Fortschritt an. Darauf klicken, um
                        dort weiterzumachen, wo Sie aufgehört haben.
                    </p>
                    <p>
                        <strong>Deck-Verwaltung:</strong> Verwenden Sie die Aktionsschaltflächen, um Decks umzubenennen, zu synchronisieren, zurückzusetzen oder zu löschen. FSRS-Parameter (Retentionsziel, max. Intervall, Fuzz) können pro Deck
                        in den Einstellungen (Zahnrad-Symbol) konfiguriert werden.
                    </p>

                    <h3>Turniere</h3>
                    <p>Turniere ermöglichen das Gruppieren von Matches nach Veranstaltung. Das Tournament-Panel mit <strong>Ctrl+Y</strong> öffnen, um Turniere zu verwalten und ihnen Matches zuzuordnen.</p>

                    <h3>Stats</h3>
                    <p>
                        Das Stats-Panel (<strong>Ctrl+D</strong>) zeigt Leistungsstatistiken (PR und MWC-Kosten), die aus allen importierten Positionen berechnet werden. Verwenden Sie die Filterleiste, um die Analyse nach
                        Spieler, Turnier, Datumsbereich, Entscheidungstyp oder Matchlänge einzuschränken. Auf einen beliebigen Indikator klicken, um zu den entsprechenden Positionen zu gelangen.
                    </p>
`,
    shortcuts: `
                    <h3>Datenbank</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Tastenkürzel</th>
                                <th>Beschreibung</th>
                            </tr>
                        </thead>

                        <tbody>
                            <tr>
                                <td>Ctrl + N</td>
                                <td>Neue Datenbank</td>
                            </tr>

                            <tr>
                                <td>Ctrl + O</td>
                                <td>Datenbank öffnen</td>
                            </tr>

                            <tr>
                                <td>Ctrl + Shift + I</td>
                                <td>Datenbank importieren</td>
                            </tr>

                            <tr>
                                <td>Ctrl + Shift + S</td>
                                <td>Datenbank exportieren</td>
                            </tr>

                            <tr>
                                <td>Ctrl + Q</td>
                                <td>blunderDB beenden</td>
                            </tr>

                            <tr>
                                <td>Ctrl + M</td>
                                <td>Metadaten bearbeiten</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Position</h3>

                    <table>
                        <thead>
                            <tr>
                                <th>Tastenkürzel</th>
                                <th>Beschreibung</th>
                            </tr>
                        </thead>

                        <tbody>
                            <tr>
                                <td>Ctrl + I</td>
                                <td>Position oder Match importieren</td>
                            </tr>

                            <tr>
                                <td>Ctrl + C</td>
                                <td>Position kopieren (kopiert auch in die Brett-Zwischenablage)</td>
                            </tr>

                            <tr>
                                <td>Ctrl + X</td>
                                <td>Brettbild in die Zwischenablage kopieren (PNG)</td>
                            </tr>

                            <tr>
                                <td>Ctrl + X, Ctrl + X</td>
                                <td>Bild Brett + Analyse in die Zwischenablage kopieren (PNG)</td>
                            </tr>

                            <tr>
                                <td>Ctrl + V</td>
                                <td>Position einfügen (im Suchpanel: auf das Brett einfügen)</td>
                            </tr>

                            <tr>
                                <td>Ctrl + S</td>
                                <td>Position speichern</td>
                            </tr>

                            <tr>
                                <td>Ctrl + U</td>
                                <td>Position aktualisieren</td>
                            </tr>

                            <tr>
                                <td>Del</td>
                                <td>Position löschen</td>
                            </tr>

                            <tr>
                                <td>Backspace</td>
                                <td>Brett, Cube, Spielstand und Würfel zurücksetzen</td>
                            </tr>

                            <tr>
                                <td>Ctrl + G</td>
                                <td>Positionsmetadaten anzeigen</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Navigation</h3>

                    <table>
                        <thead>
                            <tr>
                                <th>Tastenkürzel</th>
                                <th>Beschreibung</th>
                            </tr>
                        </thead>

                        <tbody>
                            <tr>
                                <td>Ctrl + R</td>
                                <td>Alle Positionen neu laden</td>
                            </tr>

                            <tr>
                                <td>PageUp, h</td>
                                <td>Erste Position / Vorherige Partie (Match-Navigation)</td>
                            </tr>

                            <tr>
                                <td>Left, k</td>
                                <td>Vorherige Position</td>
                            </tr>

                            <tr>
                                <td>Right, j</td>
                                <td>Nächste Position</td>
                            </tr>

                            <tr>
                                <td>PageDown, l</td>
                                <td>Letzte Position / Nächste Partie (Match-Navigation)</td>
                            </tr>

                            <tr>
                                <td>r</td>
                                <td>Zufällige Position laden</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Anzeige</h3>

                    <table>
                        <thead>
                            <tr>
                                <th>Tastenkürzel</th>
                                <th>Beschreibung</th>
                            </tr>
                        </thead>

                        <tbody>
                            <tr>
                                <td>Ctrl + ArrowLeft</td>
                                <td>Brettausrichtung nach links setzen</td>
                            </tr>

                            <tr>
                                <td>Ctrl + ArrowRight</td>
                                <td>Brettausrichtung nach rechts setzen</td>
                            </tr>

                            <tr>
                                <td>p</td>
                                <td>Pip-Count ein-/ausblenden</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Aktionen</h3>

                    <table>
                        <thead>
                            <tr>
                                <th>Tastenkürzel</th>
                                <th>Beschreibung</th>
                            </tr>
                        </thead>

                        <tbody>
                            <tr>
                                <td>Tab</td>
                                <td>Suchpanel öffnen (Positionseditor)</td>
                            </tr>

                            <tr>
                                <td>Space</td>
                                <td>Befehlszeile öffnen</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Werkzeuge</h3>

                    <table>
                        <thead>
                            <tr>
                                <th>Tastenkürzel</th>
                                <th>Beschreibung</th>
                            </tr>
                        </thead>

                        <tbody>
                            <tr>
                                <td>Ctrl + L</td>
                                <td>Analyse anzeigen</td>
                            </tr>

                            <tr>
                                <td>Ctrl + P</td>
                                <td>Kommentare schreiben</td>
                            </tr>

                            <tr>
                                <td>Ctrl + K</td>
                                <td>Anki-Panel anzeigen</td>
                            </tr>

                            <tr>
                                <td>Ctrl + F</td>
                                <td>Suchpanel</td>
                            </tr>

                            <tr>
                                <td>Ctrl + Tab</td>
                                <td>Match-Panel</td>
                            </tr>

                            <tr>
                                <td>Ctrl + B</td>
                                <td>Collection-Panel</td>
                            </tr>

                            <tr>
                                <td>Ctrl + Y</td>
                                <td>Tournaments-Panel</td>
                            </tr>

                            <tr>
                                <td>Ctrl + D</td>
                                <td>Stats-Panel</td>
                            </tr>

                            <tr>
                                <td>Ctrl + E</td>
                                <td>EPC-Panel</td>
                            </tr>

                            <tr>
                                <td>?</td>
                                <td>Hilfe öffnen</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Befehlszeile</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Tastenkürzel</th>
                                <th>Beschreibung</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>Up</td>
                                <td>Im Befehlsverlauf nach oben blättern</td>
                            </tr>
                            <tr>
                                <td>Down</td>
                                <td>Im Befehlsverlauf nach unten blättern</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Analyse-Panel</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Tastenkürzel</th>
                                <th>Beschreibung</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>Click</td>
                                <td>Einen Zug auswählen/abwählen (Pfeile ein-/ausblenden)</td>
                            </tr>
                            <tr>
                                <td>Up, k</td>
                                <td>Vorherigen Zug auswählen (wenn ein Zug ausgewählt ist)</td>
                            </tr>
                            <tr>
                                <td>Down, j</td>
                                <td>Nächsten Zug auswählen (wenn ein Zug ausgewählt ist)</td>
                            </tr>
                            <tr>
                                <td>d</td>
                                <td>Zwischen Stein- und Cube-Analyse umschalten (nur Match-Navigation)</td>
                            </tr>
                            <tr>
                                <td>Esc</td>
                                <td>Zug abwählen. Wenn kein Zug ausgewählt ist, das Panel schließen.</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Search History-Panel</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Tastenkürzel</th>
                                <th>Beschreibung</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>Click</td>
                                <td>Eine Suche auswählen/abwählen (Position anzeigen)</td>
                            </tr>
                            <tr>
                                <td>Double-click</td>
                                <td>Suche ausführen und Panel schließen</td>
                            </tr>
                            <tr>
                                <td>Up, k</td>
                                <td>Vorherige Suche auswählen (neuer, oben)</td>
                            </tr>
                            <tr>
                                <td>Down, j</td>
                                <td>Nächste Suche auswählen (älter, unten)</td>
                            </tr>
                            <tr>
                                <td>Esc</td>
                                <td>Suche abwählen. Wenn keine Suche ausgewählt ist, das Panel schließen.</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Filter Library-Panel</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Tastenkürzel</th>
                                <th>Beschreibung</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>Click</td>
                                <td>Einen Filter auswählen/abwählen (Position anzeigen)</td>
                            </tr>
                            <tr>
                                <td>Double-click</td>
                                <td>Filtersuche ausführen</td>
                            </tr>
                            <tr>
                                <td>Up, k</td>
                                <td>Vorherigen Filter auswählen (wenn ein Filter ausgewählt ist)</td>
                            </tr>
                            <tr>
                                <td>Down, j</td>
                                <td>Nächsten Filter auswählen (wenn ein Filter ausgewählt ist)</td>
                            </tr>
                            <tr>
                                <td>Esc</td>
                                <td>Filter abwählen. Wenn kein Filter ausgewählt ist, das Panel schließen.</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Match-Panel</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Tastenkürzel</th>
                                <th>Beschreibung</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>Click</td>
                                <td>Ein Match auswählen/abwählen</td>
                            </tr>
                            <tr>
                                <td>Double-click</td>
                                <td>Match-Positionen laden</td>
                            </tr>
                            <tr>
                                <td>Up, k</td>
                                <td>Vorheriges Match auswählen</td>
                            </tr>
                            <tr>
                                <td>Down, j</td>
                                <td>Nächstes Match auswählen</td>
                            </tr>
                            <tr>
                                <td>Del</td>
                                <td>Ausgewähltes Match löschen</td>
                            </tr>
                            <tr>
                                <td>Esc</td>
                                <td>Match abwählen. Wenn kein Match ausgewählt ist, das Panel schließen.</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Collection-Panel</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Tastenkürzel</th>
                                <th>Beschreibung</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>Click</td>
                                <td>Eine Sammlung auswählen (ihre Positionen anzeigen)</td>
                            </tr>
                            <tr>
                                <td>Double-click</td>
                                <td>Sammlung öffnen und ihre Positionen durchsehen</td>
                            </tr>
                            <tr>
                                <td>Del</td>
                                <td>Aktuelle Position aus der aktiven Sammlung entfernen</td>
                            </tr>
                            <tr>
                                <td>Esc</td>
                                <td>Sammlung abwählen. Wenn keine Sammlung ausgewählt ist, das Panel schließen.</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Anki-Panel</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Tastenkürzel</th>
                                <th>Beschreibung</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>1</td>
                                <td>Bewertung: Again (Wiederholung fehlgeschlagen, bald erneut anzeigen)</td>
                            </tr>
                            <tr>
                                <td>2</td>
                                <td>Bewertung: Hard (schwieriges Erinnern)</td>
                            </tr>
                            <tr>
                                <td>3</td>
                                <td>Bewertung: Good (korrektes Erinnern)</td>
                            </tr>
                            <tr>
                                <td>4</td>
                                <td>Bewertung: Easy (müheloses Erinnern)</td>
                            </tr>
                            <tr>
                                <td>Esc</td>
                                <td>Wiederholung stoppen und zur Deck-Liste zurückkehren (später fortsetzbar)</td>
                            </tr>
                        </tbody>
                    </table>
`,
    commands: `
                    <h3>Datenbank</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Befehl</th>
                                <th>Beschreibung</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>new, ne, n</td>
                                <td>Eine neue Datenbank erstellen</td>
                            </tr>
                            <tr>
                                <td>open, op, o</td>
                                <td>Eine bestehende Datenbank öffnen</td>
                            </tr>
                            <tr>
                                <td>import_db, idb</td>
                                <td>Eine andere Datenbank importieren und zusammenführen</td>
                            </tr>
                            <tr>
                                <td>export_db, edb</td>
                                <td>Aktuelle Auswahl in eine neue Datenbank exportieren</td>
                            </tr>
                            <tr>
                                <td>quit, q</td>
                                <td>blunderDB beenden</td>
                            </tr>
                            <tr>
                                <td>meta</td>
                                <td>Metadaten bearbeiten</td>
                            </tr>
                        </tbody>
                    </table>
                    <h3>Position</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Befehl</th>
                                <th>Beschreibung</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>import, i</td>
                                <td>Eine Position oder ein Match importieren</td>
                            </tr>
                            <tr>
                                <td>write, wr, w</td>
                                <td>Eine Position speichern</td>
                            </tr>
                            <tr>
                                <td>write!, wr!, w!</td>
                                <td>Eine Position aktualisieren</td>
                            </tr>
                            <tr>
                                <td>delete, del, d</td>
                                <td>Eine Position löschen</td>
                            </tr>
                        </tbody>
                    </table>
                    <h3>Navigation</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Befehl</th>
                                <th>Beschreibung</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>[number]</td>
                                <td>Zu einer bestimmten Position per Index springen</td>
                            </tr>
                        </tbody>
                    </table>
                    <h3>Werkzeuge</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Befehl</th>
                                <th>Beschreibung</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>list, l</td>
                                <td>Analyse anzeigen</td>
                            </tr>
                            <tr>
                                <td>comment, co</td>
                                <td>Kommentare schreiben</td>
                            </tr>
                            <tr>
                                <td>filter, fl</td>
                                <td>Filterbibliothek anzeigen</td>
                            </tr>
                            <tr>
                                <td>history, hi</td>
                                <td>Suchverlauf anzeigen</td>
                            </tr>
                            <tr>
                                <td>match, ma</td>
                                <td>Match-Panel anzeigen</td>
                            </tr>
                            <tr>
                                <td>collection, coll</td>
                                <td>Collections-Panel anzeigen</td>
                            </tr>
                            <tr>
                                <td>epc</td>
                                <td>EPC-Rechner (Effective Pip Count)</td>
                            </tr>
                            <tr>
                                <td>m</td>
                                <td>Zuletzt besuchtes Match navigieren</td>
                            </tr>
                            <tr>
                                <td>help, he, h</td>
                                <td>Hilfe öffnen</td>
                            </tr>
                            <tr>
                                <td>met</td>
                                <td>Die Kazaross-XG2 Match Equity Table öffnen</td>
                            </tr>
                            <tr>
                                <td>tp2</td>
                                <td>Take Point beim 2er-Cube (Live und Last)</td>
                            </tr>
                            <tr>
                                <td>tp2_live</td>
                                <td>Take Point beim 2er-Cube in langen Rennen</td>
                            </tr>
                            <tr>
                                <td>tp2_last</td>
                                <td>Take Point beim 2er-Cube in Last-Roll-Positionen</td>
                            </tr>
                            <tr>
                                <td>tp4</td>
                                <td>Take Point beim 4er-Cube (Live und Last)</td>
                            </tr>
                            <tr>
                                <td>tp4_live</td>
                                <td>Take Point beim 4er-Cube in langen Rennen</td>
                            </tr>
                            <tr>
                                <td>tp4_last</td>
                                <td>Take Point beim 4er-Cube in Last-Roll-Positionen</td>
                            </tr>
                            <tr>
                                <td>gv1</td>
                                <td>Gammon-Werte beim 1er-Cube</td>
                            </tr>
                            <tr>
                                <td>gv2</td>
                                <td>Gammon-Werte beim 2er-Cube</td>
                            </tr>
                            <tr>
                                <td>gv4</td>
                                <td>Gammon-Werte beim 4er-Cube</td>
                            </tr>
                            <tr>
                                <td>#tag1 tag2 ...</td>
                                <td>Position taggen</td>
                            </tr>
                            <tr>
                                <td>s</td>
                                <td>Positionen mit Filtern suchen</td>
                            </tr>
                            <tr>
                                <td>ss</td>
                                <td>In aktuellen Ergebnissen mit Filtern suchen</td>
                            </tr>
                            <tr>
                                <td>e</td>
                                <td>Alle Positionen neu laden</td>
                            </tr>
                        </tbody>
                    </table>
                    <h3>Filter</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Filter</th>
                                <th>Beschreibung</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>cube, cub, cu, c</td>
                                <td>Cube einbeziehen</td>
                            </tr>
                            <tr>
                                <td>score, sco, sc, s</td>
                                <td>Spielstand einbeziehen</td>
                            </tr>
                            <tr>
                                <td>d</td>
                                <td>Entscheidungstyp einbeziehen</td>
                            </tr>
                            <tr>
                                <td>D</td>
                                <td>Würfelwurf einbeziehen</td>
                            </tr>
                            <tr>
                                <td>nc</td>
                                <td>Kontaktlose Positionen einbeziehen</td>
                            </tr>
                            <tr>
                                <td>M</td>
                                <td>Gespiegelte Positionen einbeziehen</td>
                            </tr>
                            <tr>
                                <td>i</td>
                                <td>Nur Positionen, die einzeln importiert wurden, nicht durch einen Match-Import</td>
                            </tr>
                            <tr>
                                <td>x</td>
                                <td
                                    >Positionen ausschließen, die <em>irgendeinen</em> Stein der im Tab „Except" gezeichneten Struktur enthalten (z. B. behält das Zeichnen von Steinen auf 1, 3 und 5 nur Positionen, die keinen davon
                                    enthalten). „At least" / „Except" über den Filtern umschalten, um die ausgeschlossenen Steine auf dem Brett zu zeichnen (mit einem roten Hinweis dargestellt). Die Anzahl pro Feld ist nicht begrenzt (3 auf einem Feld
                                    schließt dort 3+ aus — ein gemachter Punkt ohne Reserve), und zwei schnelle Klicks auf ein Feld markieren es als leer erforderlich (eine rot schraffierte Zelle, beliebige Farbe); ein einzelner Klick auf dieses
                                    Feld gibt es wieder frei. Auf einem gemeinsamen Feld hat „Except" Vorrang vor „At least", wenn sich beide widersprechen.</td
                                >
                            </tr>
                            <tr>
                                <td>p>x</td>
                                <td>Pip-Count &gt; x</td>
                            </tr>
                            <tr>
                                <td>p&lt;x</td>
                                <td>Pip-Count &lt; x</td>
                            </tr>
                            <tr>
                                <td>px,y</td>
                                <td>Pip-Count zwischen x und y</td>
                            </tr>
                            <tr>
                                <td>P>x</td>
                                <td>Absoluter Pip-Count des Spielers &gt; x</td>
                            </tr>
                            <tr>
                                <td>P&lt;x</td>
                                <td>Absoluter Pip-Count des Spielers &lt; x</td>
                            </tr>
                            <tr>
                                <td>Px,y</td>
                                <td>Absoluter Pip-Count des Spielers zwischen x und y</td>
                            </tr>
                            <tr>
                                <td>e>x</td>
                                <td>Equity &gt; x (in Millipoints)</td>
                            </tr>
                            <tr>
                                <td>e&lt;x</td>
                                <td>Equity &lt; x (in Millipoints)</td>
                            </tr>
                            <tr>
                                <td>ex,y</td>
                                <td>Equity zwischen x und y (in Millipoints)</td>
                            </tr>
                            <tr>
                                <td>E>x</td>
                                <td>Zugfehler von Spieler 1 &gt; x (in Millipoints)</td>
                            </tr>
                            <tr>
                                <td>E&lt;x</td>
                                <td>Zugfehler von Spieler 1 &lt; x (in Millipoints)</td>
                            </tr>
                            <tr>
                                <td>Ex,y</td>
                                <td>Zugfehler von Spieler 1 zwischen x und y (in Millipoints)</td>
                            </tr>
                            <tr>
                                <td>w>x</td>
                                <td>Gewinnrate &gt; x</td>
                            </tr>
                            <tr>
                                <td>w&lt;x</td>
                                <td>Gewinnrate &lt; x</td>
                            </tr>
                            <tr>
                                <td>wx,y</td>
                                <td>Gewinnrate zwischen x und y</td>
                            </tr>
                            <tr>
                                <td>g>x</td>
                                <td>Gammon-Rate &gt; x</td>
                            </tr>
                            <tr>
                                <td>g&lt;x</td>
                                <td>Gammon-Rate &lt; x</td>
                            </tr>
                            <tr>
                                <td>gx,y</td>
                                <td>Gammon-Rate zwischen x und y</td>
                            </tr>
                            <tr>
                                <td>b>x</td>
                                <td>Backgammon-Rate &gt; x</td>
                            </tr>
                            <tr>
                                <td>b&lt;x</td>
                                <td>Backgammon-Rate &lt; x</td>
                            </tr>
                            <tr>
                                <td>bx,y</td>
                                <td>Backgammon-Rate zwischen x und y</td>
                            </tr>
                            <tr>
                                <td>W>x</td>
                                <td>Gewinnrate des Gegners &gt; x</td>
                            </tr>
                            <tr>
                                <td>W&lt;x</td>
                                <td>Gewinnrate des Gegners &lt; x</td>
                            </tr>
                            <tr>
                                <td>Wx,y</td>
                                <td>Gewinnrate des Gegners zwischen x und y</td>
                            </tr>
                            <tr>
                                <td>G>x</td>
                                <td>Gammon-Rate des Gegners &gt; x</td>
                            </tr>
                            <tr>
                                <td>G&lt;x</td>
                                <td>Gammon-Rate des Gegners &lt; x</td>
                            </tr>
                            <tr>
                                <td>Gx,y</td>
                                <td>Gammon-Rate des Gegners zwischen x und y</td>
                            </tr>
                            <tr>
                                <td>B>x</td>
                                <td>Backgammon-Rate des Gegners &gt; x</td>
                            </tr>
                            <tr>
                                <td>B&lt;x</td>
                                <td>Backgammon-Rate des Gegners &lt; x</td>
                            </tr>
                            <tr>
                                <td>Bx,y</td>
                                <td>Backgammon-Rate des Gegners zwischen x und y</td>
                            </tr>
                            <tr>
                                <td>o>x</td>
                                <td>Ausgewürfelte Steine des Spielers &gt; x</td>
                            </tr>
                            <tr>
                                <td>o&lt;x</td>
                                <td>Ausgewürfelte Steine des Spielers &lt; x</td>
                            </tr>
                            <tr>
                                <td>ox,y</td>
                                <td>Ausgewürfelte Steine des Spielers zwischen x und y</td>
                            </tr>
                            <tr>
                                <td>O>x</td>
                                <td>Ausgewürfelte Steine des Gegners &gt; x</td>
                            </tr>
                            <tr>
                                <td>O&lt;x</td>
                                <td>Ausgewürfelte Steine des Gegners &lt; x</td>
                            </tr>
                            <tr>
                                <td>Ox,y</td>
                                <td>Ausgewürfelte Steine des Gegners zwischen x und y</td>
                            </tr>
                            <tr>
                                <td>k>x</td>
                                <td>Hintere Steine des Spielers &gt; x</td>
                            </tr>
                            <tr>
                                <td>k&lt;x</td>
                                <td>Hintere Steine des Spielers &lt; x</td>
                            </tr>
                            <tr>
                                <td>kx,y</td>
                                <td>Hintere Steine des Spielers zwischen x und y</td>
                            </tr>
                            <tr>
                                <td>K>x</td>
                                <td>Hintere Steine des Gegners &gt; x</td>
                            </tr>
                            <tr>
                                <td>K&lt;x</td>
                                <td>Hintere Steine des Gegners &lt; x</td>
                            </tr>
                            <tr>
                                <td>Kx,y</td>
                                <td>Hintere Steine des Gegners zwischen x und y</td>
                            </tr>
                            <tr>
                                <td>z>x</td>
                                <td>Steine des Spielers in der Zone &gt; x</td>
                            </tr>
                            <tr>
                                <td>z&lt;x</td>
                                <td>Steine des Spielers in der Zone &lt; x</td>
                            </tr>
                            <tr>
                                <td>zx,y</td>
                                <td>Steine des Spielers in der Zone zwischen x und y</td>
                            </tr>
                            <tr>
                                <td>Z>x</td>
                                <td>Steine des Gegners in der Zone &gt; x</td>
                            </tr>
                            <tr>
                                <td>Z&lt;x</td>
                                <td>Steine des Gegners in der Zone &lt; x</td>
                            </tr>
                            <tr>
                                <td>Zx,y</td>
                                <td>Steine des Gegners in der Zone zwischen x und y</td>
                            </tr>
                            <tr>
                                <td>bo>x</td>
                                <td>Außenfeld-Blot des Spielers &gt; x</td>
                            </tr>
                            <tr>
                                <td>bo&lt;x</td>
                                <td>Außenfeld-Blot des Spielers &lt; x</td>
                            </tr>
                            <tr>
                                <td>box,y</td>
                                <td>Außenfeld-Blot des Spielers zwischen x und y</td>
                            </tr>
                            <tr>
                                <td>BO>x</td>
                                <td>Außenfeld-Blot des Gegners &gt; x</td>
                            </tr>
                            <tr>
                                <td>BO&lt;x</td>
                                <td>Außenfeld-Blot des Gegners &lt; x</td>
                            </tr>
                            <tr>
                                <td>BOx,y</td>
                                <td>Außenfeld-Blot des Gegners zwischen x und y</td>
                            </tr>
                            <tr>
                                <td>bj&gt;x</td>
                                <td>Blot im Heimbrett des Spielers &gt; x</td>
                            </tr>
                            <tr>
                                <td>bj&lt;x</td>
                                <td>Blot im Heimbrett des Spielers &lt; x</td>
                            </tr>
                            <tr>
                                <td>bjx,y</td>
                                <td>Blot im Heimbrett des Spielers zwischen x und y</td>
                            </tr>
                            <tr>
                                <td>BJ&gt;x</td>
                                <td>Blot im Heimbrett des Gegners &gt; x</td>
                            </tr>
                            <tr>
                                <td>BJ&lt;x</td>
                                <td>Blot im Heimbrett des Gegners &lt; x</td>
                            </tr>
                            <tr>
                                <td>BJx,y</td>
                                <td>Blot im Heimbrett des Gegners zwischen x und y</td>
                            </tr>

                            <tr>
                                <td>t"word1;word2;..."</td>
                                <td>Textsuche</td>
                            </tr>
                            <tr>
                                <td>m"pattern1;pattern2;..."</td>
                                <td>Beste Züge, die mindestens eines der angegebenen Muster enthalten</td>
                            </tr>
                            <tr>
                                <td>m"ND;DT;DP;..."</td>
                                <td>Beste Cube-Entscheidung aus No Double/Take, Double/Take, Double/Pass</td>
                            </tr>
                            <tr>
                                <td>T&gt;x</td>
                                <td>Erstellungsdatum &gt; x (Jahr/Monat/Tag)</td>
                            </tr>
                            <tr>
                                <td>T&lt;x</td>
                                <td>Erstellungsdatum &lt; x (Jahr/Monat/Tag)</td>
                            </tr>
                            <tr>
                                <td>Tx,y</td>
                                <td>Erstellungsdatum zwischen x und y</td>
                            </tr>
                            <tr>
                                <td>max</td>
                                <td>Im Match mit ID x suchen (z. B. ma3)</td>
                            </tr>
                            <tr>
                                <td>max,y</td>
                                <td>In Matches mit IDs von x bis y suchen (z. B. ma2,5)</td>
                            </tr>
                            <tr>
                                <td>tnx</td>
                                <td>Im Turnier mit ID x suchen (z. B. tn1)</td>
                            </tr>
                            <tr>
                                <td>tnx,y</td>
                                <td>In Turnieren mit IDs von x bis y suchen (z. B. tn1,3)</td>
                            </tr>
                        </tbody>
                    </table>
                    <h3>Sonstiges</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Befehl</th>
                                <th>Beschreibung</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>clear, cl</td>
                                <td>Befehlsverlauf löschen</td>
                            </tr>
                            <tr>
                                <td>migrate_from_1_0_to_1_1</td>
                                <td>Datenbank von Version 1.0 auf 1.1 migrieren</td>
                            </tr>
                            <tr>
                                <td>migrate_from_1_1_to_1_2</td>
                                <td>Datenbank von Version 1.1 auf 1.2 migrieren</td>
                            </tr>
                            <tr>
                                <td>migrate_from_1_2_to_1_3</td>
                                <td>Datenbank von Version 1.2 auf 1.3 migrieren</td>
                            </tr>
                        </tbody>
                    </table>
`,
    about: `
                    <h3>Version</h3>
                    <p>Application version: {appVersion}</p>
                    <p>Database version: {dbVersion}</p>

                    <h3>Autor</h3>
                    <p><strong>Kévin Unger &lt;blunderdb@proton.me&gt;</strong></p>
                    <p>Sie finden mich auch auf Heroes unter dem Spitznamen <strong>postmanpat</strong>.</p>
                    <p>
                        Ich habe blunderDB ursprünglich für meinen persönlichen Gebrauch entwickelt, um Muster in meinen Fehlern zu erkennen. Aber es ist sehr angenehm, Rückmeldungen zu bekommen, besonders wenn viele Stunden
                        in Konzeption, Programmierung, Fehlersuche... investiert wurden. Zögern Sie also nicht, mir zu schreiben, um Ihre Eindrücke zu teilen.
                    </p>
                    <p>Hier sind mehrere Möglichkeiten, mich zu erreichen:</p>
                    <ul>
                        <li>Sprechen Sie mit mir, wenn wir uns in einem Turnier treffen,</li>
                        <li>Schicken Sie mir eine E-Mail,</li>
                    </ul>
                    <h3>Lizenz</h3>
                    <p>
                        blunderDB ist unter der MIT-Lizenz lizenziert. Das bedeutet, dass es Ihnen freisteht, die Software zu nutzen, zu kopieren, zu ändern, zusammenzuführen, zu veröffentlichen, zu verteilen, unterzulizenzieren und/oder Kopien der Software zu verkaufen, vorausgesetzt,
                        dass der ursprüngliche Copyright-Hinweis und dieser Berechtigungshinweis in allen Kopien oder wesentlichen Teilen der Software enthalten sind.
                    </p>
                    <h3>Danksagungen</h3>
                    <p>Ich widme diese kleine Software meiner Partnerin <strong>Anne-Claire</strong> und unserer lieben Tochter <strong>Perrine</strong>. Ganz besonders möchte ich einigen Freunden danken:</p>
                    <ul>
                        <li>
                            <strong>Tristan Remille</strong>, dafür, dass er mich mit Freude und Freundlichkeit in das Backgammon eingeführt hat; dafür, dass er mir den Weg zum Verständnis dieses wunderbaren Spiels gezeigt hat; dafür, dass er mich
                            trotz meiner schwachen Versuche, besser zu spielen, weiterhin unterstützt.
                        </li>
                        <li><strong>Nicolas Harmand</strong>, ein fröhlicher Begleiter seit über einem Jahrzehnt bei großartigen Abenteuern und ein fantastischer Spielpartner, seit er vom Backgammon-Virus erfasst wurde.</li>
                    </ul>
                    <p>Die Kazaross-XG2 Match Equity Table (MET) wird <strong>Neil Kazaross</strong> zugeschrieben.</p>
                    <p>Die Take-Point- und Gammon-Wert-Tabellen stammen aus dem Buch <em>The Theory of Backgammon</em> von <strong>Dirk Schiemann</strong>.</p>
                    <p>
                        Die einseitige 6-Punkte-Auswürfeldatenbank, die für die EPC-Berechnung (Effective Pip Count) verwendet wird, wurde mit <strong>GNU Backgammon</strong> (GnuBG) erzeugt. GnuBG ist ein freies und quelloffenes
                        Backgammon-Programm, lizenziert unter der GNU General Public License.
                    </p>
`
};
