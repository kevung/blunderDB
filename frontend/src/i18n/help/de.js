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
<h3>Einführung</h3>
<p>
    blunderDB ist eine Software zum Erstellen von Backgammon-Positionsdatenbanken. Ihre größte Stärke besteht darin, einen einzigen Ort zu bieten, an dem ein Spieler die von ihm erlebten Positionen
    (online, in Turnieren) zusammenführen und diese Positionen erneut studieren kann, indem er sie nach verschiedenen, beliebig kombinierbaren Filtern filtert. blunderDB kann auch verwendet werden, um
    Kataloge von Referenzpositionen zu erstellen.
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
    <li>Matches in Turnieren organisieren,</li>
    <li>über ein Terminal Positionen ohne Analyse stapelweise mit dem eingebetteten gammonNet-Auswerter analysieren (Befehl <strong>analyze</strong> von blunderDB).</li>
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
    <li>EPC-Werte für Auswürfelpositionen zu berechnen (Eval-Panel),</li>
    <li>gespeicherte Suchfilter durchzusehen (Filter Library-Panel),</li>
    <li>den Suchverlauf durchzusehen (Search History-Panel).</li>
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
    Durch Drücken der <strong>Tab</strong>-Taste wird das Suchpanel geöffnet, und es kann eine Position auf dem Brett bearbeitet werden, um sie der Datenbank hinzuzufügen oder eine zu suchende
    Positionsstruktur zu definieren. Die Verteilung der Steine, der Cube, der Spielstand und das Zugrecht können mit der Maus geändert werden.
</p>

<h3>Befehlszeile</h3>
<p>
    Die in die Statusleiste integrierte Befehlszeile ermöglicht alle Funktionen von blunderDB: Datenbankoperationen, Positionsnavigation, Anzeigen von Analysen und Kommentaren, Suche nach Positionen
    mit Filtern... Sobald die Oberfläche vertraut ist, wird empfohlen, schrittweise die Befehlszeile zu verwenden, die eine leistungsstarke und flüssige Nutzung von blunderDB ermöglicht, insbesondere
    für die Positionssuchfunktionen.
</p>
<p>
    Zum Öffnen der Befehlszeile die <strong>Space</strong>-Taste drücken. In der Statusleiste erscheint eine Eingabeaufforderung. Befehl eingeben und zur Ausführung <strong>Enter</strong> drücken.
    <strong>Escape</strong>
    drücken, um abzubrechen.
</p>
<p>
    blunderDB führt die vom Benutzer gesendeten Abfragen aus, sofern sie gültig sind, und ändert bei Bedarf sofort den Zustand der Datenbank. Es sind keine expliziten Speicheraktionen durch den
    Benutzer erforderlich.
</p>
<p>
    Um eine Suche innerhalb zuvor gefilterter Positionen zu verfeinern, den Befehl <strong>ss</strong> gefolgt von Filtern verwenden (z. B. <strong>ss nc</strong>). Dies schränkt die Suche auf nur die
    aktuell angezeigten Positionen ein und ermöglicht eine schrittweise Eingrenzung der Ergebnisse. Das Suchpanel (<strong>Ctrl+F</strong>) bietet ebenfalls ein Kontrollkästchen „In aktuellen
    Ergebnissen suchen" für dieselbe Funktion.
</p>

<h3>EPC-Rechner</h3>
<p>Der EPC-Rechner (Effective Pip Count) berechnet den effektiven Pip-Count von Auswürfelpositionen. Er verwendet die einseitige 6-Punkte-Auswürfeldatenbank von GnuBG für exakte EPC-Werte.</p>
<p>
    Zum Öffnen des Eval-Panels <strong>Ctrl+E</strong> drücken, im unteren Panel auf den Eval-Tab klicken oder <strong>epc</strong> in die Befehlszeile eingeben. Das Brett wird mit einer
    Standard-Auswürfelkonfiguration (15 Steine) initialisiert.
</p>
<p>
    Sie können Steine auf den Feldern des Heimbretts mit der Maus frei hinzufügen oder entfernen. Die EPC-Werte werden in Echtzeit im dafür vorgesehenen Eval-Panel angezeigt und zeigen für jeden
    Spieler:
</p>
<ul>
    <li><strong>EPC</strong>: die durchschnittliche Anzahl der Pips, die zum Auswürfeln aller Steine benötigt werden,</li>
    <li><strong>Pip Count</strong>: der reine Pip-Count,</li>
    <li><strong>Wastage</strong>: die Differenz zwischen EPC und Pip-Count,</li>
    <li><strong>Avg Rolls</strong>: durchschnittliche Anzahl der Würfe zum Auswürfeln,</li>
    <li><strong>Std Dev</strong>: Standardabweichung der Anzahl der Würfe.</li>
</ul>
<p>Wenn beide Spieler Steine in ihrem Heimbrett haben, zeigt ein Vergleichsbereich die EPC- und Pip-Count-Differenzen.</p>
<p>Zum Schließen des Eval-Panels erneut <strong>Ctrl+E</strong> drücken oder zu einem anderen Tab wechseln.</p>
<p>
    Bei einer reinen Auswürfelposition zeigt eine Renntabelle zusätzlich die Gewinnchancen beider Spieler und, wenn die Position von einer Two-Sided-Datenbank abgedeckt ist (eingebaut bis 6 Steine pro
    Spieler, erweiterte Datenbank bis 11 über den Bearoff-Tab der Einstellungen herunterladbar), die exakten Money-Equities — mit dem Equity-Abstand jeder nicht optimalen Entscheidung zur besten —
    sowie die beste Würfel-Entscheidung. Außerhalb dieses Bereichs wird die Gewinnchance geschätzt (Badge „geschätzt" mit Fehlermarge) und keine Entscheidung angezeigt. Der Spieler am Zug wird durch
    Klick auf das Auswürfel-/Punkterechteck eines Spielers geändert, die Würfelposition durch Klick auf den Würfel.
</p>
<p>
    Das Kontrollkästchen <strong>Challenge</strong> verdeckt die Ergebnisse nach jeder Änderung; ein Klick auf eine Zone deckt sie auf — ideal, um EPC und Würfel-Entscheidung erst zu schätzen und dann
    zu prüfen.
</p>

<h3>Match-Navigation</h3>
<p>
    blunderDB ermöglicht das Durchsehen der Züge importierter Matches. Das Match-Panel mit <strong>Ctrl+Tab</strong> öffnen und auf ein Match doppelklicken (oder <strong>Enter</strong> drücken), um
    dessen Positionen zu laden.
</p>
<p>
    Bei der Navigation in einem Match wird die zuletzt besuchte Position automatisch gespeichert und wiederhergestellt. Mit den Tasten <strong>Left</strong>/<strong>Right</strong> zwischen Positionen
    wechseln und mit <strong>PageUp</strong>/<strong>PageDown</strong> zwischen Partien springen.
</p>
<p>
    Das Analyse-Panel (<strong>Ctrl+L</strong>) zeigt die Analyse für jeden Zug, wobei der gespielte Zug hervorgehoben wird. <strong>d</strong> drücken, um zwischen Stein- und Cube-Analyse
    umzuschalten.
</p>

<h3>Sammlungen</h3>
<p>
    Sammlungen ermöglichen das Organisieren von Positionen in benutzerdefinierten Gruppen. Das Collection-Panel mit <strong>Ctrl+B</strong> öffnen und dann auf eine Sammlung doppelklicken, um deren
    Positionen durchzusehen. Sammlungen und die darin enthaltenen Positionen können per Drag-and-drop umgeordnet werden.
</p>

<h3>Anki (verteilte Wiederholung)</h3>
<p>Das Anki-Panel (<strong>Ctrl+K</strong>) bietet verteilte Wiederholung zum Studieren von Backgammon-Positionen mit dem FSRS-Algorithmus.</p>
<p>
    <strong>Decks erstellen:</strong> Auf <em>New Deck</em> klicken, um ein Deck aus einer Sammlung oder aus den aktuellen Suchergebnissen zu erstellen. Suchbasierte Decks werden automatisch
    synchronisiert, wenn der Anki-Tab aktiviert wird.
</p>
<p>
    <strong>Wiederholen:</strong> Ein Deck auswählen und auf <em>Study</em> klicken (oder auf ein Deck doppelklicken), um mit dem Wiederholen fälliger Karten zu beginnen. Jede Karte zeigt die
    entsprechende Position auf dem Brett. Bewerten Sie Ihr Erinnerungsvermögen mit den Tasten <strong>1</strong> (Again), <strong>2</strong> (Hard), <strong>3</strong> (Good) oder
    <strong>4</strong> (Easy). <strong>Esc</strong> drücken, um zu stoppen und zur Deck-Liste zurückzukehren.
</p>
<p>
    <strong>Sitzung begrenzen:</strong> In den Deck-Einstellungen können Sie eine Sitzung auf eine Kartenzahl begrenzen. Die Sitzung endet dann mit einem entsprechenden Hinweis; das freie Üben bleibt
    verfügbar, ohne den Plan zu verändern. Eine Begrenzung auf <em>0</em> zeigt keine Karte — was nicht dasselbe ist wie keine Begrenzung.
</p>
<p>
    <strong>Behaltensrate:</strong> Die Zielrate ist Ihre Wahl beim Kompromiss zwischen Aufwand und Qualität. Daneben zeigen die Einstellungen die <em>gemessene</em> Rate aus Ihren Wiederholungen —
    eine Information, keine Steuerung. Eine Änderung wirkt nicht rückwirkend: Jede Karte übernimmt den neuen Rhythmus bei ihrer nächsten Wiederholung.
</p>
<p>
    <strong>Antwort anzeigen:</strong> Die Karte stellt eine Frage; überlegen Sie, und drücken Sie dann die <strong>Leertaste</strong> (oder klicken Sie auf den verdeckten Bereich), um die
    gespeicherte Analyse der Stellung aufzudecken. Sie erscheint unter den Bewertungstasten, die in Reichweite bleiben. Zum Bewerten müssen Sie sie nicht aufdecken; bei der nächsten Karte wird sie
    wieder verdeckt — nicht jedoch, wenn Sie nur den Reiter wechseln.
</p>
<p>
    <strong>Stoppen/Fortsetzen:</strong> Sie können eine Wiederholungssitzung jederzeit durch Drücken von <strong>Esc</strong> stoppen. Die Schaltfläche wechselt zu <em>Resume</em> und zeigt Ihren
    Fortschritt an. Darauf klicken, um dort weiterzumachen, wo Sie aufgehört haben.
</p>
<p>
    <strong>Deck-Verwaltung:</strong> Verwenden Sie die Aktionsschaltflächen, um Decks umzubenennen, zu synchronisieren, zurückzusetzen oder zu löschen. FSRS-Parameter (Retentionsziel, max. Intervall,
    Fuzz) können pro Deck in den Einstellungen (Zahnrad-Symbol) konfiguriert werden.
</p>

<h3>Turniere</h3>
<p>Turniere ermöglichen das Gruppieren von Matches nach Veranstaltung. Das Tournament-Panel mit <strong>Ctrl+Y</strong> öffnen, um Turniere zu verwalten und ihnen Matches zuzuordnen.</p>

<h3>Stats</h3>
<p>
    Das Stats-Panel (<strong>Ctrl+D</strong>) zeigt Leistungsstatistiken (PR und MWC-Kosten), die aus allen importierten Positionen berechnet werden. Verwenden Sie die Filterleiste, um die Analyse
    nach Spieler, Turnier, Datumsbereich, Entscheidungstyp oder Matchlänge einzuschränken. Auf einen beliebigen Indikator klicken, um zu den entsprechenden Positionen zu gelangen. Der Tab
    <strong>Spieler</strong> listet pro Spieler die Anzahl der Matches, die Bilanz, die Entscheidungen, den PR (Steine und Dopplerwürfel), den Snowie, die Blunders und das über die bekannten Würfe
    gemessene Glück auf.
</p>

<h3>Wasserzeichen und geschützter Export</h3>
<p>Beim Exportieren (<strong>export_db</strong> oder dem Dialog Exportieren) lassen sich zwei unabhängige Schutzmechanismen frei aktivieren, der eine, der andere oder beide zugleich:</p>
<ul>
    <li>
        <strong>Wasserzeichen:</strong> kennzeichnet die exportierte Datei mit ihrer Herkunft (wer sie erstellt hat, eine optionale Notiz). Das Wasserzeichen ist mit Ihrer Ausstelleridentität
        signiert: Es kann nicht verändert oder in fremdem Namen gefälscht werden — es ist aber nicht unentfernbar und verhindert keine Kopie.
    </li>
    <li>
        <strong>Passwort:</strong> legt den Export in einen verschlüsselten <strong>.dbx</strong>-Container. Es schützt die Datei auf dem Transportweg, nicht die Datenbank selbst — wem Sie das
        Passwort geben, kann sie öffnen — und die Herkunft bleibt auch ohne Passwort lesbar.
    </li>
</ul>
<p>
    Ihre Ausstelleridentität, der Schlüssel, der Ihre Wasserzeichen signiert, entsteht automatisch beim ersten Export, der mit seiner Herkunft gekennzeichnet wird. Sehen Sie sie ein, exportieren oder
    regenerieren Sie sie über den Tab <strong>Ausstelleridentität</strong> der Einstellungen.
</p>
`,
    shortcuts: `
<div class="admonition note">
<p>Die Tastenkürzel sind unabhängig von der Tastaturbelegung: Sie bleiben unabhängig von der verwendeten Belegung (AZERTY, QWERTY, QWERTZ usw.) auf die gleiche Weise erreichbar.</p>
</div>
<div class="admonition note">
<p>Befindet sich der Cursor in einem Eingabefeld (Kommentar, Suchfeld, Befehlszeile), gelten die üblichen Tastenkürzel zur Textbearbeitung für den Text und nicht für die Stellung: CTRL-C, CTRL-X und CTRL-V kopieren, schneiden aus und fügen die Auswahl ein, CTRL-A wählt sie vollständig aus, CTRL-Z und CTRL-Y machen rückgängig und stellen wieder her.</p>
</div>
<h3>Datenbank</h3>
<table>
<thead>
<tr>
<th>Tastenkürzel</th>
<th>Aktion</th>
</tr>
</thead>
<tbody>
<tr>
<td>STRG-N</td>
<td>Eine neue Datenbank erstellen.</td>
</tr>
<tr>
<td>STRG-O</td>
<td>Eine bestehende Datenbank öffnen.</td>
</tr>
<tr>
<td>STRG-UMSCHALT-I</td>
<td>Eine Datenbank importieren.</td>
</tr>
<tr>
<td>STRG-UMSCHALT-S</td>
<td>Die Datenbank exportieren.</td>
</tr>
<tr>
<td>STRG-Q</td>
<td>blunderDB schließen.</td>
</tr>
<tr>
<td>STRG-M</td>
<td>Die Metadaten der Datenbank bearbeiten.</td>
</tr>
</tbody>
</table>
<h3>Position</h3>
<table>
<thead>
<tr>
<th>Tastenkürzel</th>
<th>Aktion</th>
</tr>
</thead>
<tbody>
<tr>
<td>STRG-I</td>
<td>Eine oder mehrere Positionen/Matches aus einer Datei importieren (xg, xgp, sgf, mat, txt, bgf).</td>
</tr>
<tr>
<td>STRG-UMSCHALT-F</td>
<td>Einen Ordner mit Match-/Positionsdateien rekursiv importieren.</td>
</tr>
<tr>
<td>STRG-C</td>
<td>Eine Position in die Zwischenablage kopieren.</td>
</tr>
<tr>
<td>STRG-X</td>
<td>Das Bild des Boards in die Zwischenablage kopieren (PNG).</td>
</tr>
<tr>
<td>STRG-X STRG-X</td>
<td>Das Bild des Boards mit Analyse in die Zwischenablage kopieren (PNG).</td>
</tr>
<tr>
<td>STRG-V</td>
<td>Eine Position aus der Zwischenablage einfügen (automatische Formaterkennung).</td>
</tr>
<tr>
<td>STRG-S</td>
<td>Eine Position speichern.</td>
</tr>
<tr>
<td>STRG-U</td>
<td>Eine Position aktualisieren.</td>
</tr>
<tr>
<td>Entf</td>
<td>Aktuelle Stellung löschen (Bestätigung erforderlich).</td>
</tr>
<tr>
<td>RÜCKTASTE</td>
<td>Board, Dopplerwürfel, Spielstand und Würfel zurücksetzen.</td>
</tr>
<tr>
<td>STRG-G</td>
<td>Die Metadaten der Position anzeigen.</td>
</tr>
</tbody>
</table>
<div class="admonition note">
<p>Im Eval-Panel (siehe Eval-Panel) setzt RÜCKTASTE auf die panel-eigenen Werte zurück (Money-Spielstand, keine gelegten Würfel) statt auf die des Bearbeitungsmodus (Punktestand 7 überall, Würfel 3-1). Ein Doppelklick außerhalb des Bretts löst dieselbe Zurücksetzung aus.</p>
</div>
<div class="admonition note">
<p>Im Eval-Panel und im Suchpanel ist das Brett ein Entwurf und keine Stellung der Datenbank: CTRL-V <strong>legt die Stellung dort auf das Brett</strong>, statt sie in die Datenbank zu importieren, und CTRL-C kopiert das angezeigte Brett — seine XGID wird aus den gesetzten Steinen neu berechnet, ohne die Analyse der zuvor betrachteten Stellung. Die kopierte Stellung lässt sich so unverändert in eXtreme Gammon oder in eine andere blunderDB-Instanz einfügen.</p>
</div>
<h3>Navigation</h3>
<table>
<thead>
<tr>
<th>Tastenkürzel</th>
<th>Aktion</th>
</tr>
</thead>
<tbody>
<tr>
<td>STRG-R</td>
<td>Alle Positionen aus der Datenbank neu laden.</td>
</tr>
<tr>
<td>Bild-auf, h</td>
<td>Erste Position / Vorheriges Spiel (Match-Navigation).</td>
</tr>
<tr>
<td>LINKS, k</td>
<td>Vorherige Position.</td>
</tr>
<tr>
<td>RECHTS, j</td>
<td>Nächste Position.</td>
</tr>
<tr>
<td>OBEN, k</td>
<td>Vorheriger Zug (wenn ein Zug in der Analyse ausgewählt ist).</td>
</tr>
<tr>
<td>UNTEN, j</td>
<td>Nächster Zug (wenn ein Zug in der Analyse ausgewählt ist).</td>
</tr>
<tr>
<td>Bild-ab, l</td>
<td>Letzte Position / Nächstes Spiel (Match-Navigation).</td>
</tr>
<tr>
<td>r</td>
<td>Eine zufällige Position laden.</td>
</tr>
</tbody>
</table>
<h3>Anzeige</h3>
<table>
<thead>
<tr>
<th>Tastenkürzel</th>
<th>Aktion</th>
</tr>
</thead>
<tbody>
<tr>
<td>STRG-LINKS</td>
<td>Board-Ausrichtung nach links.</td>
</tr>
<tr>
<td>STRG-RECHTS</td>
<td>Board-Ausrichtung nach rechts.</td>
</tr>
<tr>
<td>p</td>
<td>Pipcount ein-/ausblenden.</td>
</tr>
</tbody>
</table>
<h3>Aktionen</h3>
<table>
<thead>
<tr>
<th>Tastenkürzel</th>
<th>Aktion</th>
</tr>
</thead>
<tbody>
<tr>
<td>TAB</td>
<td>Das Suchfenster öffnen (Positionseditor).</td>
</tr>
<tr>
<td>LEERTASTE</td>
<td>Die Befehlszeile öffnen.</td>
</tr>
</tbody>
</table>
<div class="admonition note">
<p>TAB öffnet das Suchpanel nur, wenn der Fokus auf dem Spielbrett liegt (oder nirgends im Besonderen, was meistens der Fall ist). Sobald der Fokus auf einer Schaltfläche, einem Eingabefeld oder einem Link liegt, setzt TAB die normale Tastaturnavigation zwischen den Oberflächenelementen fort, anstatt dieses Panel erneut zu öffnen.</p>
</div>
<h3>Werkzeuge</h3>
<table>
<thead>
<tr>
<th>Tastenkürzel</th>
<th>Aktion</th>
</tr>
</thead>
<tbody>
<tr>
<td>STRG-L</td>
<td>Die Analyse ein-/ausblenden.</td>
</tr>
<tr>
<td>STRG-P</td>
<td>Die Kommentare ein-/ausblenden.</td>
</tr>
<tr>
<td>STRG-K</td>
<td>Das Anki-Fenster ein-/ausblenden (verteiltes Wiederholen).</td>
</tr>
<tr>
<td>STRG-F</td>
<td>Das Suchfenster ein-/ausblenden.</td>
</tr>
<tr>
<td>STRG-Tab</td>
<td>Das Match-Fenster ein-/ausblenden.</td>
</tr>
<tr>
<td>STRG-B</td>
<td>Das Sammlungen-Fenster ein-/ausblenden.</td>
</tr>
<tr>
<td>STRG-Y</td>
<td>Das Turnier-Fenster ein-/ausblenden.</td>
</tr>
<tr>
<td>STRG-D</td>
<td>Das Statistik-Fenster ein-/ausblenden.</td>
</tr>
<tr>
<td>STRG-E</td>
<td>Das Eval-Panel ein-/ausblenden.</td>
</tr>
<tr>
<td>?</td>
<td>Die Hilfe ein-/ausblenden.</td>
</tr>
</tbody>
</table>
<div class="admonition note">
<p>Die Reihenfolge dieser Tabs wird, sobald sie per Drag &amp; Drop auf der Leiste geändert wurde, sitzungsübergreifend gespeichert. Ein Rechtsklick auf einen Tab ermöglicht es, ihn auszublenden; die dann rechts in der Leiste erscheinende Pfeil-Schaltfläche öffnet ein Menü, um ausgeblendete Tabs wieder einzublenden — sie bleiben in der Zwischenzeit über ihre Tastenkombination erreichbar.</p>
</div>
<h3>Ansichts-Reiter</h3>
<table>
<thead>
<tr>
<th>Tastenkürzel</th>
<th>Aktion</th>
</tr>
</thead>
<tbody>
<tr>
<td>STRG-T</td>
<td>Eine neue Ansicht erstellen (Kopie der aktuellen Ansicht).</td>
</tr>
<tr>
<td>STRG-W</td>
<td>Die aktuelle Ansicht schließen.</td>
</tr>
<tr>
<td>STRG-Bild-auf, UMSCHALT-J</td>
<td>Vorherige Ansicht.</td>
</tr>
<tr>
<td>STRG-Bild-ab, UMSCHALT-K</td>
<td>Nächste Ansicht.</td>
</tr>
<tr>
<td>STRG-1 … STRG-9</td>
<td>Direkt zur n-ten Ansicht springen.</td>
</tr>
<tr>
<td>Doppelklick auf den Reiter</td>
<td>Die Ansicht umbenennen.</td>
</tr>
</tbody>
</table>
<div class="admonition note">
<p>Die Richtung von UMSCHALT-J / UMSCHALT-K ist gegenüber j / k umgekehrt: <em>j</em> geht vorwärts (nächste Stellung) und <em>k</em> zurück (vorherige Stellung), während <em>UMSCHALT-J</em> zur vorherigen Ansicht zurückkehrt und <em>UMSCHALT-K</em> zur nächsten wechselt. Das ist beabsichtigt (kein zu korrigierendes Tastenkürzel) — UMSCHALT-J/UMSCHALT-K folgen der Konvention von STRG-BildAuf/STRG-BildAb, mit der sie verbunden sind, nicht der von j/k.</p>
</div>
<h3>Befehlszeile</h3>
<table>
<thead>
<tr>
<th>Tastenkürzel</th>
<th>Aktion</th>
</tr>
</thead>
<tbody>
<tr>
<td>OBEN</td>
<td>Im Befehlsverlauf nach oben blättern.</td>
</tr>
<tr>
<td>UNTEN</td>
<td>Im Befehlsverlauf nach unten blättern.</td>
</tr>
</tbody>
</table>
<h3>Suchverlauf</h3>
<p>Der Suchverlauf ist der Reiter <em>Verlauf</em> des Suchfensters (<em>CTRL-F</em> oder <em>TAB</em>).</p>
<table>
<thead>
<tr>
<th>Tastenkürzel</th>
<th>Aktion</th>
</tr>
</thead>
<tbody>
<tr>
<td>Klick</td>
<td>Eine Suche auswählen/abwählen (Position anzeigen).</td>
</tr>
<tr>
<td>Doppelklick</td>
<td>Die Suche ausführen.</td>
</tr>
</tbody>
</table>
<h3>Filterbibliothek</h3>
<p>Die Filterbibliothek ist der Reiter <em>Gespeichert</em> des Suchfensters (<em>CTRL-F</em> oder <em>TAB</em>).</p>
<table>
<thead>
<tr>
<th>Tastenkürzel</th>
<th>Aktion</th>
</tr>
</thead>
<tbody>
<tr>
<td>Klick</td>
<td>Einen Filter auswählen/abwählen (Position anzeigen).</td>
</tr>
<tr>
<td>Doppelklick</td>
<td>Die Suche des Filters ausführen.</td>
</tr>
</tbody>
</table>
<h3>Analyse-Fenster</h3>
<table>
<thead>
<tr>
<th>Tastenkürzel</th>
<th>Aktion</th>
</tr>
</thead>
<tbody>
<tr>
<td>Klick</td>
<td>Einen Zug auswählen/abwählen (Pfeile ein-/ausblenden).</td>
</tr>
<tr>
<td>OBEN, k</td>
<td>Vorherigen Zug auswählen (wenn ein Zug ausgewählt ist).</td>
</tr>
<tr>
<td>UNTEN, j</td>
<td>Nächsten Zug auswählen (wenn ein Zug ausgewählt ist).</td>
</tr>
<tr>
<td>d</td>
<td>Zwischen Zug- und Dopplerwürfel-Analyse wechseln (nur Match-Navigation).</td>
</tr>
<tr>
<td>Esc</td>
<td>Den Zug abwählen. Wenn kein Zug ausgewählt ist, das Fenster schließen.</td>
</tr>
</tbody>
</table>
<h3>Eval-Fenster</h3>
<p>Die Zugliste des Fensters <em>Eval</em> (siehe Eval-Panel) wird wie die des Analyse-Fensters durchlaufen. Diese Tastenkürzel wirken, sobald ein Zug per Klick ausgewählt wurde.</p>
<table>
<thead>
<tr>
<th>Tastenkürzel</th>
<th>Aktion</th>
</tr>
</thead>
<tbody>
<tr>
<td>Klick</td>
<td>Einen Zug auswählen/abwählen (Pfeile ein-/ausblenden).</td>
</tr>
<tr>
<td>OBEN, k</td>
<td>Vorherigen Zug auswählen (wenn ein Zug ausgewählt ist).</td>
</tr>
<tr>
<td>UNTEN, j</td>
<td>Nächsten Zug auswählen (wenn ein Zug ausgewählt ist).</td>
</tr>
<tr>
<td>Esc</td>
<td>Den Zug abwählen.</td>
</tr>
</tbody>
</table>
<h3>Match-Fenster</h3>
<table>
<thead>
<tr>
<th>Tastenkürzel</th>
<th>Aktion</th>
</tr>
</thead>
<tbody>
<tr>
<td>Klick</td>
<td>Ein Match auswählen.</td>
</tr>
<tr>
<td>Doppelklick</td>
<td>Im Match navigieren.</td>
</tr>
<tr>
<td>OBEN, k</td>
<td>Vorheriges Match auswählen.</td>
</tr>
<tr>
<td>UNTEN, j</td>
<td>Nächstes Match auswählen.</td>
</tr>
<tr>
<td>EINGABE</td>
<td>Das ausgewählte Match laden.</td>
</tr>
<tr>
<td>Entf</td>
<td>Das ausgewählte Match löschen.</td>
</tr>
<tr>
<td>Esc</td>
<td>Abwählen/Fenster schließen.</td>
</tr>
</tbody>
</table>
<h3>Anki-Fenster (verteiltes Wiederholen)</h3>
<table>
<thead>
<tr>
<th>Tastenkürzel</th>
<th>Aktion</th>
</tr>
</thead>
<tbody>
<tr>
<td>LEERTASTE, Klick</td>
<td>Die Antwort anzeigen (die gespeicherte Analyse der Position).</td>
</tr>
<tr>
<td>1</td>
<td>Bewerten: Nochmal (nicht gewusst, bald wiederholen).</td>
</tr>
<tr>
<td>2</td>
<td>Bewerten: Schwer.</td>
</tr>
<tr>
<td>3</td>
<td>Bewerten: Gut.</td>
</tr>
<tr>
<td>4</td>
<td>Bewerten: Einfach.</td>
</tr>
<tr>
<td>p</td>
<td>Pip-Count ein-/ausblenden (wie das allgemeine Tastenkürzel, während der Wiederholung verfügbar).</td>
</tr>
<tr>
<td>Esc</td>
<td>Die Wiederholung beenden und zur Stapelliste zurückkehren (später fortsetzbar).</td>
</tr>
</tbody>
</table>
<h3>Turnier-Fenster</h3>
<table>
<thead>
<tr>
<th>Tastenkürzel</th>
<th>Aktion</th>
</tr>
</thead>
<tbody>
<tr>
<td>Klick, Doppelklick</td>
<td>Ein Turnier auswählen (Details anzeigen).</td>
</tr>
<tr>
<td>OBEN, k</td>
<td>Vorheriges Turnier auswählen.</td>
</tr>
<tr>
<td>UNTEN, j</td>
<td>Nächstes Turnier auswählen.</td>
</tr>
<tr>
<td>Doppelklick (auf ein Match des Turniers)</td>
<td>Im Match navigieren.</td>
</tr>
<tr>
<td>Esc</td>
<td>Die laufende Bearbeitung abbrechen, sonst die Suche zum Hinzufügen eines Matches leeren, sonst die Turnierauswahl aufheben, sonst das Fenster schließen (schrittweise).</td>
</tr>
</tbody>
</table>
<h3>Sammlungs-Fenster</h3>
<table>
<thead>
<tr>
<th>Tastenkürzel</th>
<th>Aktion</th>
</tr>
</thead>
<tbody>
<tr>
<td>Klick</td>
<td>Die aktuelle Stellung zur überfahrenen Sammlung hinzufügen/daraus entfernen.</td>
</tr>
<tr>
<td>Doppelklick</td>
<td>Die Sammlung öffnen.</td>
</tr>
<tr>
<td>Entf</td>
<td>Die aktuelle Stellung (oder die angehakten Stellungen) aus der geöffneten Sammlung entfernen.</td>
</tr>
<tr>
<td>Esc</td>
<td>Zur Liste der Sammlungen zurückkehren, sonst die Sammlungsauswahl aufheben, sonst das Fenster schließen (schrittweise).</td>
</tr>
</tbody>
</table>
<div class="admonition note">
<p>Dieses Fenster fängt die Navigationskürzel nicht ab: BildAuf/h, LINKS/k, RECHTS/j, BildAb/l bewegen sich durch die Stellungen der geöffneten Sammlung genau wie in Navigation beschrieben.</p>
</div>
`,
    commands: `
<p>Die Kommandozeile in der Statusleiste öffnet sich durch Drücken der <em>LEERTASTE</em>. Während der Eingabe eines Befehls erscheint automatisch eine Liste mit Vorschlägen: Die <em>TAB</em>-Taste (oder <em>UMSCHALT-TAB</em>) durchläuft die Vorschläge und vervollständigt den Befehl, während <em>ESC</em> die Liste schließt (ein zweites <em>ESC</em> schließt die Kommandozeile). Die Tasten <em>AUF</em> und <em>AB</em> bleiben dem Befehlsverlauf vorbehalten.</p>
<h3>Globale Operationen</h3>
<table>
<thead>
<tr>
<th>Befehl</th>
<th>Aktion</th>
</tr>
</thead>
<tbody>
<tr>
<td>new, ne, n</td>
<td>Erstellt eine neue Datenbank.</td>
</tr>
<tr>
<td>open, op, o</td>
<td>Öffnet eine vorhandene Datenbank.</td>
</tr>
<tr>
<td>import_db, idb</td>
<td>Importiert eine andere Datenbank und führt sie zusammen.</td>
</tr>
<tr>
<td>export_db, edb</td>
<td>Exportiert die aktuelle Auswahl in eine neue Datenbank.</td>
</tr>
<tr>
<td>quit, q</td>
<td>Schließt blunderDB.</td>
</tr>
<tr>
<td>help, he, h</td>
<td>Öffnet die Hilfe von blunderDB.</td>
</tr>
<tr>
<td>tutorial, tour</td>
<td>Öffnet den Katalog der geführten Touren durch die Oberfläche.</td>
</tr>
<tr>
<td>demo</td>
<td>Lädt eine Beispieldatenbank (Matches, Turnier, Sammlungen, Kommentare, Anki-Stapel, Analysen), um das Werkzeug zu erkunden.</td>
</tr>
<tr>
<td>meta</td>
<td>Zeigt die Metadaten der Datenbank an.</td>
</tr>
<tr>
<td>epc</td>
<td>Öffnet das Eval-Panel (Effective Pip Count, Gewinnwahrscheinlichkeit und Würfel-Urteil im Bearoff).</td>
</tr>
<tr>
<td>met</td>
<td>Öffnet die Match-Equity-Tabelle Kazaross-XG2.</td>
</tr>
<tr>
<td>tp2</td>
<td>Öffnet die Takepoint-Tabelle bei Dopplerstand 2.</td>
</tr>
<tr>
<td>tp2_live</td>
<td>Öffnet die Takepoint-Tabelle bei Dopplerstand 2 für lange Rennen.</td>
</tr>
<tr>
<td>tp2_last</td>
<td>Öffnet die Takepoint-Tabelle bei Dopplerstand 2 für den letzten Wurf.</td>
</tr>
<tr>
<td>tp4</td>
<td>Öffnet die Takepoint-Tabelle bei Dopplerstand 4.</td>
</tr>
<tr>
<td>tp4_live</td>
<td>Öffnet die Takepoint-Tabelle bei Dopplerstand 4 für lange Rennen.</td>
</tr>
<tr>
<td>tp4_last</td>
<td>Öffnet die Takepoint-Tabelle bei Dopplerstand 4 für den letzten Wurf.</td>
</tr>
<tr>
<td>gv1</td>
<td>Öffnet die Gammon-Wert-Tabelle bei Dopplerstand 1.</td>
</tr>
<tr>
<td>gv2</td>
<td>Öffnet die Gammon-Wert-Tabelle bei Dopplerstand 2.</td>
</tr>
<tr>
<td>gv4</td>
<td>Öffnet die Gammon-Wert-Tabelle bei Dopplerstand 4.</td>
</tr>
</tbody>
</table>
<h3>Positionen und Navigation</h3>
<table>
<thead>
<tr>
<th>Befehl</th>
<th>Aktion</th>
</tr>
</thead>
<tbody>
<tr>
<td>import, i</td>
<td>Importiert eine oder mehrere Positionen/Matches aus einer Datei (xg, xgp, sgf, mat, txt, bgf).</td>
</tr>
<tr>
<td>delete, del, d</td>
<td>Löscht die aktuelle Position (Bestätigung erforderlich).</td>
</tr>
<tr>
<td>[number]</td>
<td>Springt zur Position mit dem angegebenen Index.</td>
</tr>
<tr>
<td>list, l</td>
<td>Zeigt die Analyse der aktuellen Position an.</td>
</tr>
<tr>
<td>comment, co</td>
<td>Kommentare anzeigen/schreiben.</td>
</tr>
<tr>
<td>history, hi</td>
<td>Das Suchpanel öffnen (der Suchverlauf befindet sich in dessen Reiter <em>Verlauf</em>).</td>
</tr>
<tr>
<td>stats, st</td>
<td>Statistik-Panel anzeigen/ausblenden.</td>
</tr>
<tr>
<td>match, ma</td>
<td>Match-Panel anzeigen/ausblenden.</td>
</tr>
<tr>
<td>collection, coll</td>
<td>Sammlungs-Panel anzeigen/ausblenden.</td>
</tr>
<tr>
<td>#tag1 tag2 ...</td>
<td>Versieht die aktuelle Position mit Tags.</td>
</tr>
<tr>
<td>e</td>
<td>Lädt alle Positionen aus der Datenbank.</td>
</tr>
<tr>
<td>blunders, bl [n]</td>
<td>Lädt die größten Fehler (Equity/MWC) in die Analyseansicht, gemäß dem aktuellen Statistikfilter. Eine optionale Zahl wählt, wie viele geladen werden (<code>bl 50</code>); standardmäßig 10.Lädt die größten Fehler (Equity/MWC) in die Analyseansicht, gemäß dem aktuellen Statistikfilter.</td>
</tr>
<tr>
<td>m</td>
<td>Navigiert zum zuletzt besuchten Match.</td>
</tr>
</tbody>
</table>
<h3>Bearbeitung und Suche</h3>
<table>
<thead>
<tr>
<th>Befehl</th>
<th>Aktion</th>
</tr>
</thead>
<tbody>
<tr>
<td>write, wr, w</td>
<td>Speichert die aktuelle Position.</td>
</tr>
<tr>
<td>write!, wr!, w!</td>
<td>Aktualisiert die aktuelle Position.</td>
</tr>
<tr>
<td>s</td>
<td>Sucht nach Positionen mit Filtern.</td>
</tr>
<tr>
<td>ss</td>
<td>Sucht unter den aktuell gefilterten Positionen.</td>
</tr>
</tbody>
</table>
<h3>Suchfilter</h3>
<p>Die folgenden Filter müssen bei einer Suche aneinandergereiht werden, das heißt nach dem Befehlsbeginn <code>s</code>.</p>
<div class="admonition warning">
<p>Bei der Positionssuche berücksichtigt blunderDB standardmäßig die aktuelle Steinstruktur und ignoriert die Stellung des Dopplers, den Spielstand und die Würfel. Um die Stellung des Dopplers, den Spielstand und die Würfel zu berücksichtigen, muss dies in der Suche ausdrücklich angegeben werden.</p>
</div>
<div class="admonition note">
<p>Der Suchbefehl <code>s</code> ist im Suchpanel verfügbar (Taste <code>TAB</code>). Mit dem Befehl <code>ss</code> kann unter den aktuell gefilterten Ergebnissen gesucht werden.</p>
</div>
<div class="admonition note">
<p>blunderDB betrachtet einen rückständigen Stein (Backchecker) als einen Stein, der sich zwischen Punkt 24 und Punkt 19 befindet.</p>
</div>
<div class="admonition note">
<p>blunderDB betrachtet die Anzahl der Steine in der Zone als die Anzahl der Steine, die sich zwischen Punkt 12 und Punkt 1 befinden.</p>
</div>
<div class="admonition note">
<p>blunderDB betrachtet das Outfield als den Bereich zwischen Punkt 18 und Punkt 7.</p>
</div>
<div class="admonition note">
<p>blunderDB betrachtet das Heimfeld als den Bereich zwischen Punkt 1 und Punkt 6.</p>
</div>
<div class="admonition tip">
<p>Die Parameter zum Filtern von Positionen können beliebig kombiniert werden.</p>
</div>
<table>
<thead>
<tr>
<th>Abfrage</th>
<th>Aktion</th>
</tr>
</thead>
<tbody>
<tr>
<td>cube, cub, cu, c</td>
<td>Die Position erfüllt die Doppler-Konfiguration.</td>
</tr>
<tr>
<td>score, sco, sc, s</td>
<td>Die Position erfüllt den Spielstand.</td>
</tr>
<tr>
<td>d</td>
<td>Die Position erfüllt den Entscheidungstyp (Stein oder Doppler).</td>
</tr>
<tr>
<td>D</td>
<td>Die Position erfüllt den Würfelwurf (beide Würfel, unabhängig von der Reihenfolge).</td>
</tr>
<tr>
<td>D1</td>
<td>Die Position erfüllt den Würfelwurf nur beim ersten Würfel (der Wert des ersten Würfels erscheint auf einem der beiden Würfel der Position).</td>
</tr>
<tr>
<td>xD65</td>
<td>Die Position wurde <strong>nicht</strong> mit dem Wurf 6-5 gespielt (unabhängig von der Reihenfolge). Der Wert wird im Token angezeigt; wiederholbar, um mehrere Würfe auszuschließen (<code>xD65 xD54</code>).</td>
</tr>
<tr>
<td>nc</td>
<td>Die Position ist kontaktlos.</td>
</tr>
<tr>
<td>M</td>
<td>Die Position oder ihr Spiegelbild erfüllt die Filter.</td>
</tr>
<tr>
<td>i</td>
<td>Die Stellung wurde einzeln importiert und nicht durch einen Match-Import eingebracht.</td>
</tr>
<tr>
<td>fl</td>
<td>Die Position wurde in der Ursprungssoftware markiert, beim Import einer eXtreme-Gammon-Partie.</td>
</tr>
<tr>
<td>x</td>
<td>Die Position enthält keinen Stein der Ausschlussstruktur (Registerkarte „Except“ des Suchpanels).</td>
</tr>
<tr>
<td>p&gt;x</td>
<td>Der Spieler liegt im Rennen mindestens x Pips zurück.</td>
</tr>
<tr>
<td>p&lt;x</td>
<td>Der Spieler liegt im Rennen höchstens x Pips zurück.</td>
</tr>
<tr>
<td>px,y</td>
<td>Der Spieler liegt im Rennen zwischen x und y Pips zurück.</td>
</tr>
<tr>
<td>P&gt;x</td>
<td>Der Spieler hat ein Rennen von mindestens x Pips.</td>
</tr>
<tr>
<td>P&lt;x</td>
<td>Der Spieler hat ein Rennen von höchstens x Pips.</td>
</tr>
<tr>
<td>Px,y</td>
<td>Der Spieler hat ein Rennen zwischen x und y Pips.</td>
</tr>
<tr>
<td>e&gt;x</td>
<td>Die Equity (in Millipunkten) der Position ist größer als x.</td>
</tr>
<tr>
<td>e&lt;x</td>
<td>Die Equity (in Millipunkten) der Position ist kleiner als x.</td>
</tr>
<tr>
<td>ex,y</td>
<td>Die Equity (in Millipunkten) der Position liegt zwischen x und y.</td>
</tr>
<tr>
<td>E&gt;x</td>
<td>Der Fehler des von Spieler 1 gespielten Zuges (in Millipunkten) ist größer als x.</td>
</tr>
<tr>
<td>E&lt;x</td>
<td>Der Fehler des von Spieler 1 gespielten Zuges (in Millipunkten) ist kleiner als x.</td>
</tr>
<tr>
<td>Ex,y</td>
<td>Der Fehler des von Spieler 1 gespielten Zuges (in Millipunkten) liegt zwischen x und y.</td>
</tr>
<tr>
<td>w&gt;x</td>
<td>Der Spieler hat Gewinnchancen von mehr als x %.</td>
</tr>
<tr>
<td>w&lt;x</td>
<td>Der Spieler hat Gewinnchancen von weniger als x %.</td>
</tr>
<tr>
<td>wx,y</td>
<td>Der Spieler hat Gewinnchancen zwischen x % und y %.</td>
</tr>
<tr>
<td>g&gt;x</td>
<td>Der Spieler hat Gammon-Chancen von mehr als x %.</td>
</tr>
<tr>
<td>g&lt;x</td>
<td>Der Spieler hat Gammon-Chancen von weniger als x %.</td>
</tr>
<tr>
<td>gx,y</td>
<td>Der Spieler hat Gammon-Chancen zwischen x % und y %.</td>
</tr>
<tr>
<td>b&gt;x</td>
<td>Der Spieler hat Backgammon-Chancen von mehr als x %.</td>
</tr>
<tr>
<td>b&lt;x</td>
<td>Der Spieler hat Backgammon-Chancen von weniger als x %.</td>
</tr>
<tr>
<td>bx,y</td>
<td>Der Spieler hat Backgammon-Chancen zwischen x % und y %.</td>
</tr>
<tr>
<td>W&gt;x</td>
<td>Der Gegner hat Gewinnchancen von mehr als x %.</td>
</tr>
<tr>
<td>W&lt;x</td>
<td>Der Gegner hat Gewinnchancen von weniger als x %.</td>
</tr>
<tr>
<td>Wx,y</td>
<td>Der Gegner hat Gewinnchancen zwischen x % und y %.</td>
</tr>
<tr>
<td>G&gt;x</td>
<td>Der Gegner hat Gammon-Chancen von mehr als x %.</td>
</tr>
<tr>
<td>G&lt;x</td>
<td>Der Gegner hat Gammon-Chancen von weniger als x %.</td>
</tr>
<tr>
<td>Gx,y</td>
<td>Der Gegner hat Gammon-Chancen zwischen x % und y %.</td>
</tr>
<tr>
<td>B&gt;x</td>
<td>Der Gegner hat Backgammon-Chancen von mehr als x %.</td>
</tr>
<tr>
<td>B&lt;x</td>
<td>Der Gegner hat Backgammon-Chancen von weniger als x %.</td>
</tr>
<tr>
<td>Bx,y</td>
<td>Der Gegner hat Backgammon-Chancen zwischen x % und y %.</td>
</tr>
<tr>
<td>o&gt;x</td>
<td>Der Spieler hat mindestens x ausgewürfelte Steine.</td>
</tr>
<tr>
<td>o&lt;x</td>
<td>Der Spieler hat höchstens x ausgewürfelte Steine.</td>
</tr>
<tr>
<td>ox,y</td>
<td>Der Spieler hat zwischen x und y ausgewürfelte Steine.</td>
</tr>
<tr>
<td>O&gt;x</td>
<td>Der Gegner hat mindestens x ausgewürfelte Steine.</td>
</tr>
<tr>
<td>O&lt;x</td>
<td>Der Gegner hat höchstens x ausgewürfelte Steine.</td>
</tr>
<tr>
<td>Ox,y</td>
<td>Der Gegner hat zwischen x und y ausgewürfelte Steine.</td>
</tr>
<tr>
<td>k&gt;x</td>
<td>Der Spieler hat mindestens x rückständige Steine.</td>
</tr>
<tr>
<td>k&lt;x</td>
<td>Der Spieler hat höchstens x rückständige Steine.</td>
</tr>
<tr>
<td>kx,y</td>
<td>Der Spieler hat zwischen x und y rückständige Steine.</td>
</tr>
<tr>
<td>K&gt;x</td>
<td>Der Gegner hat mindestens x rückständige Steine.</td>
</tr>
<tr>
<td>K&lt;x</td>
<td>Der Gegner hat höchstens x rückständige Steine.</td>
</tr>
<tr>
<td>Kx,y</td>
<td>Der Gegner hat zwischen x und y rückständige Steine.</td>
</tr>
<tr>
<td>z&gt;x</td>
<td>Der Spieler hat mindestens x Steine in der Zone.</td>
</tr>
<tr>
<td>z&lt;x</td>
<td>Der Spieler hat höchstens x Steine in der Zone.</td>
</tr>
<tr>
<td>zx,y</td>
<td>Der Spieler hat zwischen x und y Steine in der Zone.</td>
</tr>
<tr>
<td>Z&gt;x</td>
<td>Der Gegner hat mindestens x Steine in der Zone.</td>
</tr>
<tr>
<td>Z&lt;x</td>
<td>Der Gegner hat höchstens x Steine in der Zone.</td>
</tr>
<tr>
<td>Zx,y</td>
<td>Der Gegner hat zwischen x und y Steine in der Zone.</td>
</tr>
<tr>
<td>bo&gt;x</td>
<td>Der Spieler hat mindestens x Blots im Outfield.</td>
</tr>
<tr>
<td>bo&lt;x</td>
<td>Der Spieler hat höchstens x Blots im Outfield.</td>
</tr>
<tr>
<td>box,y</td>
<td>Der Spieler hat zwischen x und y Blots im Outfield.</td>
</tr>
<tr>
<td>BO&gt;x</td>
<td>Der Gegner hat mindestens x Blots im Outfield.</td>
</tr>
<tr>
<td>BO&lt;x</td>
<td>Der Gegner hat höchstens x Blots im Outfield.</td>
</tr>
<tr>
<td>BOx,y</td>
<td>Der Gegner hat zwischen x und y Blots im Outfield.</td>
</tr>
<tr>
<td>bj&gt;x</td>
<td>Der Spieler hat mindestens x Blots im Heimfeld.</td>
</tr>
<tr>
<td>bj&lt;x</td>
<td>Der Spieler hat höchstens x Blots im Heimfeld.</td>
</tr>
<tr>
<td>bjx,y</td>
<td>Der Spieler hat zwischen x und y Blots im Heimfeld.</td>
</tr>
<tr>
<td>BJ&gt;x</td>
<td>Der Gegner hat mindestens x Blots im Heimfeld.</td>
</tr>
<tr>
<td>BJ&lt;x</td>
<td>Der Gegner hat höchstens x Blots im Heimfeld.</td>
</tr>
<tr>
<td>BJx,y</td>
<td>Der Gegner hat zwischen x und y Blots im Heimfeld.</td>
</tr>
<tr>
<td>t'wort1;wort2;...'</td>
<td>Die Kommentare der Position enthalten mindestens eines der Wörter.</td>
</tr>
<tr>
<td>co</td>
<td>Die Position hat einen Kommentar, unabhängig vom Inhalt.</td>
</tr>
<tr>
<td>xco</td>
<td>Die Position hat keinen Kommentar.</td>
</tr>
<tr>
<td>m'muster1,muster2,...'</td>
<td>Die besten Steinzüge, die mindestens eines der Muster enthalten.</td>
</tr>
<tr>
<td>m'ND,DT,DP,...'</td>
<td>Die besten Doppler-Entscheidungen für No Double/Take, Double Take, Double Pass.</td>
</tr>
<tr>
<td>T&gt;x</td>
<td>Datum des Hinzufügens der Position nach x (JJJJ/MM/TT).</td>
</tr>
<tr>
<td>T&lt;x</td>
<td>Datum des Hinzufügens der Position vor x (JJJJ/MM/TT).</td>
</tr>
<tr>
<td>Tx,y</td>
<td>Datum des Hinzufügens der Position zwischen x und y (JJJJ/MM/TT).</td>
</tr>
<tr>
<td>max</td>
<td>Sucht im Match mit der ID x (z. B. ma3).</td>
</tr>
<tr>
<td>max,y</td>
<td>Sucht in den Matches mit den IDs von x bis y (z. B. ma2,5).</td>
</tr>
<tr>
<td>tnx</td>
<td>Sucht im Turnier mit der ID x (z. B. tn1).</td>
</tr>
<tr>
<td>tnx,y</td>
<td>Sucht in den Turnieren mit den IDs von x bis y (z. B. tn1,3).</td>
</tr>
<tr>
<td>idx</td>
<td>Die Position mit der Kennung x suchen (z. B. id12).</td>
</tr>
<tr>
<td>idx,y</td>
<td>Die Positionen mit den Kennungen x bis y suchen (z. B. id5,10).</td>
</tr>
<tr>
<td>pl'Name'</td>
<td>Stellungen aus einer Partie suchen, an der der genannte Spieler an einer der beiden Seiten beteiligt war (z. B. pl'Alice'). Groß-/Kleinschreibung wird ignoriert.</td>
</tr>
</tbody>
</table>
<div class="admonition note">
<p>Eine Position, die Spieler 1 mehrmals gespielt hat (derselbe Zug in zwei Matches oder zwei verschiedene Züge), wird von den Filtern <code>E&gt;x</code>, <code>E&lt;x</code> und <code>Ex,y</code> anhand des <strong>größten</strong> auf dieser Position begangenen Fehlers beurteilt. <code>E&gt;100</code> beantwortet damit die Frage „Habe ich hier jemals einen Blunder gemacht?“, und <code>E&lt;20</code> behält nur die Positionen, auf denen keiner der gespielten Züge 20 Millipunkte überschritten hat.</p>
</div>
<div class="admonition note">
<p>Das Filtern von Positionen nach dem Würfelwurf (<code>D</code> oder <code>D1</code>) bedeutet <em>erst recht</em>, die Positionen nach dem Entscheidungstyp (<code>d</code>) zu filtern. Der Filter <code>D1</code> ignoriert den Wert des zweiten Würfels: Nur der Wert des ersten Würfels wird verwendet, um Positionen zuzuordnen (auf einem der beiden Würfel des Wurfs).</p>
</div>
<div class="admonition note">
<p>Beim Filter für die relative Renndifferenz (<code>p&gt;x</code>, <code>p&lt;x</code>, <code>px,y</code>) liegt der Spieler im Rennen gegenüber dem Gegner zurück, wenn <code>x&gt;0</code>, und in Führung, wenn <code>x&lt;0</code>. Beispiel: <code>p&lt;-10</code>: Der Spieler liegt im Rennen mindestens 10 Pips in Führung. <code>p50,70</code>: Der Spieler liegt im Rennen zwischen 50 und 70 Pips zurück.</p>
</div>
<div class="admonition note">
<p>Um in mehreren nicht zusammenhängenden Matches zu suchen, wird der Filter <code>ma</code> mehrfach aneinandergereiht (z. B. <code>s ma23 ma43</code> für die Matches 23 und 43). Dasselbe Prinzip gilt für Turniere mit <code>tn</code> und für Positionen mit <code>id</code> (z. B. <code>s id5 id10</code> für die Positionen 5 und 10).</p>
</div>
<div class="admonition note">
<p>Eine Suche in einem Turnier (<code>tn</code>) entspricht einer Suche in allen Matches des betreffenden Turniers. Die IDs der Matches und Turniere sind in den ID-Spalten der entsprechenden Panels sichtbar.</p>
</div>
<p>Zum Beispiel filtert der Befehl <code>s s c p-20,-5 w&gt;60 z&gt;10 K2,3</code> alle Positionen unter Berücksichtigung der Steinstruktur, des Spielstands und des Dopplers der bearbeiteten Position, bei denen der Spieler im Rennen zwischen 20 und 5 Pips in Führung liegt, mit mindestens 60 % der Gewinnchancen, mindestens 10 Steinen in der Zone, und der Gegner zwischen 2 und 3 rückständige Steine hat.</p>
<h3>Verschiedene Befehle</h3>
<table>
<thead>
<tr>
<th>Befehl</th>
<th>Aktion</th>
</tr>
</thead>
<tbody>
<tr>
<td>clear, cl</td>
<td>Löscht den Befehlsverlauf.</td>
</tr>
</tbody>
</table>
<div class="admonition note">
<p>Datenbankmigrationen werden jetzt beim Öffnen einer Datenbank automatisch durchgeführt.</p>
</div>
`,
    about: `
<h3>Version</h3>
<p>Application version: {appVersion}</p>
<p>Database version: {dbVersion}</p>

<h3>Autor</h3>
<p><strong>Kévin Unger &lt;blunderdb@proton.me&gt;</strong></p>
<p>Sie finden mich auch auf Heroes unter dem Spitznamen <strong>postmanpat</strong>.</p>
<p>
    Ich habe blunderDB ursprünglich für meinen persönlichen Gebrauch entwickelt, um Muster in meinen Fehlern zu erkennen. Aber es ist sehr angenehm, Rückmeldungen zu bekommen, besonders wenn viele
    Stunden in Konzeption, Programmierung, Fehlersuche... investiert wurden. Zögern Sie also nicht, mir zu schreiben, um Ihre Eindrücke zu teilen.
</p>
<p>Hier sind mehrere Möglichkeiten, mich zu erreichen:</p>
<ul>
    <li>Treten Sie dem Discord-Server von blunderDB bei: <a href="https://discord.gg/DA5PpzM9En" target="_blank" rel="noopener noreferrer">discord.gg/DA5PpzM9En</a>,</li>
    <li>Sprechen Sie mit mir, wenn wir uns in einem Turnier treffen,</li>
    <li>Schicken Sie mir eine E-Mail,</li>
</ul>
<h3>Lizenz</h3>
<p>
    blunderDB ist unter der MIT-Lizenz lizenziert. Das bedeutet, dass es Ihnen freisteht, die Software zu nutzen, zu kopieren, zu ändern, zusammenzuführen, zu veröffentlichen, zu verteilen,
    unterzulizenzieren und/oder Kopien der Software zu verkaufen, vorausgesetzt, dass der ursprüngliche Copyright-Hinweis und dieser Berechtigungshinweis in allen Kopien oder wesentlichen Teilen der
    Software enthalten sind.
</p>
<h3>Danksagungen</h3>
<p>Ich widme diese kleine Software meiner Partnerin <strong>Anne-Claire</strong> und unserer lieben Tochter <strong>Perrine</strong>. Ganz besonders möchte ich einigen Freunden danken:</p>
<ul>
    <li>
        <strong>Tristan Remille</strong>, dafür, dass er mich mit Freude und Freundlichkeit in das Backgammon eingeführt hat; dafür, dass er mir den Weg zum Verständnis dieses wunderbaren Spiels
        gezeigt hat; dafür, dass er mich trotz meiner schwachen Versuche, besser zu spielen, weiterhin unterstützt.
    </li>
    <li>
        <strong>Nicolas Harmand</strong>, ein fröhlicher Begleiter seit über einem Jahrzehnt bei großartigen Abenteuern und ein fantastischer Spielpartner, seit er vom Backgammon-Virus erfasst wurde.
    </li>
</ul>
<h3>Danksagungen an Dritte</h3>
<p>blunderDB enthält Code, Daten und Schriften anderer Personen. Das Wesentliche:</p>
<ul>
    <li>
        Das neuronale Netz <strong>strehl-prob5-512-512-256-128</strong> ist das Werk von <strong>Alexander Strehl</strong> (<em>alexstrehl/backgammon-ai-engine</em>, MIT). Die Suche, das
        Doppler-Modell und die Match-Equity-Tabelle darum herum sind die eigene Konfiguration von <strong>gammonNet</strong> (<a
            href="https://github.com/kevung/gammonNet"
            target="_blank"
            rel="noopener noreferrer"
            >github.com/kevung/gammonNet</a
        >, MIT).
    </li>
    <li>Die Kazaross-XG2 Match Equity Table (MET) ist das Werk von <strong>Neil Kazaross</strong>.</li>
    <li>Die Take-Point- und Gammon-Wert-Tabellen stammen aus dem Buch <em>The Theory of Backgammon</em> von <strong>Dirk Schiemann</strong>.</li>
    <li>
        Die einseitige (6 Punkte, 15 Steine, für den EPC) und die zweiseitige (6 Punkte, 6 Steine, für Cube-Urteile im Rennen) Auswürfeldatenbank wurden mit <strong>GNU Backgammon</strong> (GnuBG)
        erzeugt. GnuBG ist freie Software unter der GPL; diese Tabellen sind von ihm erzeugte Daten und werden als solche genannt.
    </li>
    <li>Matchdateien werden von <em>xgparser</em> und <em>gnubgparser</em> (LGPL-2.1) sowie von <em>bgfparser</em> (MIT) gelesen.</li>
    <li>Auf der Go-Seite: <em>modernc.org/sqlite</em> (BSD-3-Clause), <em>pgx</em>, <em>Wails</em> und <em>go-fsrs</em> (MIT).</li>
    <li>Auf der Oberflächenseite: <em>Svelte</em>, <em>two.js</em>, <em>Chart.js</em> und <em>driver.js</em> (MIT).</li>
    <li>Die Schriften <em>Nunito</em> und <em>Noto Sans JP</em> (SIL Open Font License 1.1).</li>
</ul>
<p>
    Das vollständige Verzeichnis mit den Lizenztexten ist die Datei <strong>THIRD_PARTY.md</strong>, die mit blunderDB ausgeliefert wird (<a
        href="https://github.com/kevung/blunderDB/blob/main/THIRD_PARTY.md"
        target="_blank"
        rel="noopener noreferrer"
        >github.com/kevung/blunderDB</a
    >).
</p>
`
};
