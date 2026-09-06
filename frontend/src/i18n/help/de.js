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
<h3>Einführung</h3>
<p>blunderDB ist eine Software zum Erstellen von Datenbanken mit Backgammon-Stellungen. Ihre größte Stärke besteht darin, einen einzigen Ort zu bieten, an dem ein Spieler die Stellungen sammeln kann, die ihm begegnet sind (online, im Turnier), und sie erneut studieren kann, indem er sie nach verschiedenen beliebig kombinierbaren Filtern filtert. blunderDB kann außerdem verwendet werden, um Kataloge von Referenzstellungen zu erstellen.</p>
<p>Die Stellungen werden in einer Datenbank gespeichert, die durch eine Datei <em>.db</em> dargestellt wird. Die Desktop-Anwendung öffnet diese Datei direkt, niemals eine Netzwerkadresse: Der Servermodus (Headless-Modus (Server)) ist ein anderer Modus desselben Binärprogramms, und man wechselt von einem zum anderen durch Exportieren oder Migrieren der Datenbank, nicht indem man die Anwendung auf eine URL zeigen lässt.</p>
<h3>Wichtigste Interaktionen</h3>
<p>Die wichtigsten mit blunderDB möglichen Interaktionen sind:</p>
<ul>
<li>eine neue Stellung hinzufügen,</li>
<li>eine bestehende Stellung bearbeiten,</li>
<li>das Brettbild über <strong>CTRL-X</strong> in die Zwischenablage kopieren (PNG) oder mit der vollständigen Analyse über <strong>CTRL-X CTRL-X</strong>,</li>
<li>eine bestehende Stellung löschen,</li>
<li>eine oder mehrere Stellungen suchen,</li>
<li>Matches aus verschiedenen Quellen importieren (XG, GNUbg, BGBlitz, Jellyfish), einschließlich der Kommentare aus XG-Dateien,</li>
<li>durch die Züge eines importierten Matches navigieren,</li>
<li>Stellungen in Sammlungen organisieren,</li>
<li>Matches in Turnieren organisieren.</li>
</ul>
<p>Der Benutzer kann Stellungen frei mit Tags versehen und sie mit Kommentaren annotieren.</p>
<h3>Beschreibung der Benutzeroberfläche</h3>
<p>Die Benutzeroberfläche von blunderDB besteht von oben nach unten aus:</p>
<ul>
<li>[oben] der Werkzeugleiste, die alle wichtigen auf der Datenbank ausführbaren Operationen zusammenfasst,</li>
<li>[in der Mitte] dem Hauptanzeigebereich, der das Anzeigen oder Bearbeiten von Backgammon-Stellungen ermöglicht,</li>
<li>[unten] der Statusleiste, die verschiedene Informationen über die Datenbank oder die aktuelle Stellung anzeigt und die Befehlszeile integriert.</li>
</ul>
<p>Es können Panels angezeigt werden, um:</p>
<ul>
<li>die mit der aktuellen Stellung verknüpften Analysedaten aus eXtreme Gammon (XG), GNUbg oder BGBlitz anzuzeigen,</li>
<li>Kommentare anzuzeigen, hinzuzufügen oder zu bearbeiten,</li>
<li>Stellungen nach kombinierbaren Kriterien zu suchen und zu filtern,</li>
<li>Stellungssammlungen anzuzeigen und zu verwalten (Sammlungen-Panel),</li>
<li>die Liste der importierten Matches anzuzeigen und durch die Züge eines Matches zu navigieren (Matches-Panel),</li>
<li>Turniere anzuzeigen und zu verwalten (Turnier-Panel),</li>
<li>Leistungsstatistiken anzuzeigen (Stats-Panel),</li>
<li>den EPC (Effective Pip Count) einer Bearoff-Stellung zu berechnen (Eval-Panel),</li>
<li>Stellungen durch verteiltes Wiederholen zu studieren (Anki-Panel),</li>
<li>die Metadaten der Datenbank anzuzeigen (Metadaten-Panel).</li>
</ul>
<p>Es können modale Fenster angezeigt werden, um:</p>
<ul>
<li>die Hilfe von blunderDB anzuzeigen,</li>
<li>den Katalog der geführten Touren anzeigen (siehe Geführte Touren und Beispieldatenbank),</li>
<li>den Export der Datenbank zu konfigurieren,</li>
<li>blunderDB zu konfigurieren, insbesondere die Sprache der Benutzeroberfläche (siehe Konfiguration).</li>
</ul>
<p>Der Hauptanzeigebereich stellt dem Benutzer Folgendes bereit:</p>
<ul>
<li>ein Brett, um eine Backgammon-Stellung anzuzeigen oder zu bearbeiten,</li>
<li>den Wert und den Besitzer des Dopplers,</li>
<li>den Pip-Count jedes Spielers,</li>
<li>den Punktestand jedes Spielers,</li>
<li>die zu spielenden Würfel. Wenn auf den Würfeln keine Werte angezeigt werden, gibt die Position der Würfel an, welcher Spieler am Zug ist und dass die Stellung eine Doppler-Entscheidung ist. Wenn die Doppler-Entscheidung eine Antwort auf ein Doppel ist (Annehmen/Aufgeben), wird der angebotene Dopplerwürfel in der Mitte des Bretts mit dem angebotenen Wert angezeigt.</li>
</ul>
<p>Ein Rechtsklick auf das Brett öffnet ein Kontextmenü mit: die angezeigte Stellung im Eval-Panel bewerten, ihr Spiegelbild bewerten, das Brettbild mit seiner Analyse in die Zwischenablage kopieren (das Äquivalent von <em>STRG-X STRG-X</em>, schwerer zu entdecken), <strong>das Bild in eine Datei speichern</strong> als SVG oder PNG, eine neue Ansicht auf diese Stellung öffnen und — wenn die Stellung schon aus der Datenbank kommt — sie einem Anki-Stapel hinzufügen (verteiltes Wiederholen).</p>
<p>Die Zwischenablage ist die alltägliche Geste; Speichern ist das andere Bedürfnis — die Illustration für einen Artikel, einen Forumsbeitrag, eine Lektion. <strong>SVG</strong> wird angeboten, weil das Brett eines ist: es ist die Form, die eine Vergrößerung übersteht, die man ohne Unschärfe in ein Dokument setzt. PNG leitet sich daraus ab, ebenso wie die Kopie in die Zwischenablage: eine Darstellung, drei Ziele, also kann keines von den anderen abweichen. Dieses Menü erscheint nicht im Eval-Panel und nicht im Suchpanel, wo die rechte Taste schon die Steine der anderen Farbe setzt. Siehe Eine Stellung in das Eval-Panel bringen, um eine Stellung ins Eval-Panel zu bringen.</p>
<p>Die Statusleiste ist von links nach rechts mit den folgenden Informationen aufgebaut:</p>
<ul>
<li>die Befehlszeile, erreichbar durch Drücken der Taste <em>LEERTASTE</em>,</li>
<li>eine Informationsmeldung zu einer vom Benutzer ausgeführten Operation,</li>
<li>den Index der aktuellen Stellung, gefolgt von der Zahl der Stellungen in der aktuellen Bibliothek (oder Zug-/Partieangaben beim Durchgehen eines Matches),</li>
<li>den <strong>Bibliothekszähler</strong> — „412 Stellungen · 38 Blunder · 5 Matches“ — wo jede Zahl <strong>öffnet, was sie zählt</strong>: die Stellungen, die in der Befehlszeile vorbereitete Suche <code>E&gt;100</code> oder die Matchliste. Eine Zahl, der man nicht folgen kann, ist Dekoration. Die Blunder-Schwelle ist die der Statistik, hundert Millipunkte: Zwei Schwellen ließen dasselbe Wort zwei Dinge bedeuten.</li>
</ul>
<div class="admonition note">
<p>Bei Stellungen, die aus einer Suche des Benutzers stammen, entspricht die in der Statusleiste angegebene Anzahl der Stellungen der Anzahl der gefilterten Stellungen.</p>
</div>
<p>Der Reiter <strong>Anki</strong> trägt ein <strong>Abzeichen</strong>, wenn Karten fällig sind, über alle Stapel hinweg. Diese Zahl ist der Grund, den Reiter zu öffnen; sie hat nichts dahinter verloren. Null zeigt nichts: Ein Abzeichen mit „0“ ist Rauschen.</p>
<p>Der Befehl <code>log</code> öffnet das <strong>Aktivitätsprotokoll</strong>: die letzten zweihundert Zeilen der Protokolldatei, eine Schaltfläche zum Kopieren — das Nötige, um einer Meldung einen Bericht beizulegen — und eine weitere, um den Ordner zu öffnen. Das Protokoll wird weder gefiltert noch umformatiert: Ein aufgeräumtes Protokoll lässt sich nicht mehr zitieren.</p>
<p>In der <strong>Suchhistorie</strong> des Suchpanels erscheint jedes Token eines gespeicherten Befehls als benannter Chip — <em>Kein Kontakt</em>, <em>Zugfehler</em> — statt als nacktes Token. Der genaue Befehl bleibt im Tooltip, denn er ist es, der erneut ausgeführt wird; und ein Token, das blunderDB nicht kennt, erscheint <strong>so wie es ist</strong>, statt zum Nächstliegenden übersetzt zu werden.</p>
<h3>Ansichts-Reiter</h3>
<p>Unter der Werkzeugleiste ermöglicht eine Reiterleiste, mit mehreren <strong>Ansichten</strong> parallel zu arbeiten. Jede Ansicht ist ein unabhängiger Arbeitsbereich, der seine eigene Positionsliste, den Index der aktuellen Position, die angezeigte Position, die Analyse und den ausgewählten Zug, das aktive Panel, den aktuellen Kommentar sowie den Navigationskontext in einem Match behält. So ist es zum Beispiel möglich, eine Suche in einer Ansicht offen zu halten, während man in einer anderen ein Match durchgeht.</p>
<ul>
<li><strong>Eine Ansicht erstellen</strong>: auf die Schaltfläche <em>+</em> der Reiterleiste klicken oder <em>CTRL-T</em> drücken. Die neue Ansicht startet als Kopie der aktuellen Ansicht.</li>
<li><strong>Eine Ansicht schließen</strong>: auf das Kreuz des Reiters klicken oder <em>CTRL-W</em> drücken. Die letzte Ansicht kann nicht geschlossen werden.</li>
<li><strong>Die Ansicht wechseln</strong>: auf einen Reiter klicken, <em>CTRL-PageUp</em> / <em>CTRL-PageDown</em> (oder <em>UMSCHALT-J</em> / <em>UMSCHALT-K</em>) drücken, um zur vorherigen / nächsten Ansicht zu wechseln, oder <em>CTRL-1</em> bis <em>CTRL-9</em>, um direkt die n-te Ansicht zu erreichen.</li>
<li><strong>Eine Ansicht umbenennen</strong>: auf den Reiter doppelklicken, den neuen Namen eingeben und mit <em>EINGABE</em> bestätigen.</li>
</ul>
<p>Die Ansichten werden mit dem Sitzungszustand der Datenbank gespeichert und beim erneuten Öffnen wiederhergestellt.</p>
<h3>Konfiguration</h3>
<p>Die Einstellungsschaltfläche (Zahnradsymbol) in der Symbolleiste, links neben der Hilfeschaltfläche, öffnet das Einstellungsfenster von blunderDB. Es ist in sechs Reiter gegliedert:</p>
<ul>
<li><strong>Oberfläche</strong> — Sprache, Anzeigeskalierung, Position des Panels;</li>
<li><strong>Brettfarben</strong> — die Farben des Bretts;</li>
<li><strong>Bearoff</strong> — die vom Eval-Panel verwendeten Bearoff-Tabellen;</li>
<li><strong>gammonNet</strong> — die Einstellungen des eingebetteten Evaluators, unten beschrieben;</li>
<li><strong>Überwachter Ordner</strong> — der automatische Import von Matches, die in einem Ordner eintreffen, unten beschrieben;</li>
<li><strong>Ausstelleridentität</strong> — der Schlüssel, mit dem Ihre Wasserzeichen signiert werden; beschrieben im Abschnitt Eine Datenbank weitergeben: Herkunft und Passwort.</li>
</ul>
<p>Der Reiter <em>Oberfläche</em> beginnt mit einem <strong>Thema</strong>: <em>dem System folgen</em>, <em>hell</em>, <em>dunkel</em>, <em>hoher Kontrast</em> oder <em>druckbar</em>. Das Thema legt die Farben der Oberfläche fest und <strong>schlägt eine Brettpalette vor</strong> — eine dunkle Oberfläche um ein helles Brett ist kein dunkles Thema, sondern ein halbes, denn das Brett nimmt den Großteil des Fensters ein.</p>
<p>Sie behalten das letzte Wort, und der Mechanismus garantiert es, statt es zu versprechen: der Reiter <em>Farben</em> stellt das Brett weiterhin direkt ein, und eine nach dem Thema gewählte Farbe ist Ihre. Beim Start werden nur die Oberflächen-Token angewendet, nie die Brettpalette — die von Ihnen eingestellte ist bereits geladen, und sie bei jedem Start zu überschreiben würde Ihre Arbeit Sitzung für Sitzung löschen. Siehe <code>ADR-0038 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0038-a-named-theme-carries-the-board-palette-and-the-user-still-has-the-last-word.md&gt;</code>__.</p>
<p><em>Dem System folgen</em> ist die Voreinstellung: es folgt der Hell/Dunkel-Vorgabe des Desktops, auch wenn sie sich mitten in der Sitzung ändert. Ein Werkzeug drängt einem Desktop, der schon entschieden hat, nicht sein Hell oder sein Dunkel auf.</p>
<p>Der Reiter <em>Oberfläche</em> erlaubt außerdem die Sprachwahl unter Englisch, Französisch, Deutsch, Italienisch, Spanisch, Finnisch, Japanisch, Griechisch und Russisch. Die gesamte Oberfläche (Symbolleiste, Panels, Meldungen, Hilfe) wird in die gewählte Sprache übersetzt. Die Sprachwahl wird gespeichert und von Sitzung zu Sitzung beibehalten.</p>
<p>Derselbe Reiter bietet auch die Schaltfläche <strong>Datenbank komprimieren</strong>, die den durch Löschungen (Matches, Turniere, Bereinigungen) freigewordenen Speicherplatz zurückgewinnt: Die Datenbank schrumpft beim Löschen von Daten nie von selbst, diese Komprimierung muss ausdrücklich angefordert werden. Der Vorgang kann bei einer großen Datenbank dauern und benötigt vorübergehend etwa die doppelte Dateigröße an freiem Speicherplatz (blunderDB verweigert den Start, statt eine abgebrochene Komprimierung zu riskieren); daher wird vorher eine Bestätigung verlangt. Das Ergebnis — der gewonnene Platz in Megabyte — erscheint anschließend in der Statusleiste. Derselbe Vorgang steht auf der Kommandozeile über <code>blunderdb vacuum</code> zur Verfügung (siehe Befehlszeilenschnittstelle (CLI)).</p>
<p>Die Schaltfläche <strong>Protokollordner öffnen</strong> direkt darunter öffnet den Ordner mit dem Anwendungsprotokoll — nützlich, um einer Fehlermeldung Details beizulegen, besonders wenn blunderDB über eine Verknüpfung oder einen Doppelklick gestartet wurde, ohne angehängtes Terminal, das irgendetwas anzeigen könnte.</p>
<p>Das Kontrollkästchen <strong>Beim Start nach Updates suchen</strong>, standardmäßig aus, fragt bei jedem Start einmal die Releases-Seite des GitHub-Repositorys ab und zeigt in der Statusleiste eine Meldung, wenn eine neuere Version verfügbar ist — nie ein Fenster, das die Arbeit blockiert. Bei einer Installation über einen Paketmanager (Flatpak, Homebrew, ein Distributionspaket …) bleibt diese Prüfung automatisch deaktiviert: dann kümmert sich dieser Kanal um die Updates, nicht blunderDB selbst.</p>
<p>Auf der Registerkarte <em>Brettfarben</em> lassen sich die Farben des Bretts anpassen. Jedes Element besitzt einen eigenen Farbwähler: der Hintergrund, der Rahmen, die hellen und dunklen Zungen, die Steine von Spieler 1 und Spieler 2, die Würfel, die Würfelaugen und der Dopplerwürfel. Die Schaltfläche <em>Zurücksetzen</em> stellt alle Standardfarben wieder her. Wie die Sprache bleiben auch die gewählten Farben von einer Sitzung zur nächsten erhalten.</p>
<p>Der Reiter <em>Bearoff</em> verwaltet die Ausspieltabellen des Eval-Panels (siehe Eval-Panel). Sie sind <strong>weder in die ausführbare Datei eingebettet noch heruntergeladen</strong>: blunderDB berechnet sie auf der Maschine, die sie braucht, und das Ergebnis ist Byte für Byte dasselbe, das gnubg erzeugt — der SHA-256-Fingerabdruck wird geprüft, bevor eine Tabelle angenommen wird.</p>
<p>Die beiden gewöhnlichen Tabellen (TS-06-06 für das Verdopplungsurteil, OS-06 für den EPC) werden beim ersten Start im Hintergrund und ohne Nachfrage berechnet: etwa sechs Sekunden auf einem Kern, während derer die Anwendung normal benutzt wird. Das Eval-Panel erwähnt es nur, wenn dort eine Stellung gelegt wird, die eine noch nicht fertige Tabelle braucht.</p>
<p>Der Reiter zeigt den aktiven Bereich und seine Herkunft, den Zustand der einseitigen Tabelle, die der EPC liest, den Ordner, in dem das alles liegt, und die Liste der vorhandenen Tabellen mit ihrer Größe und ihrem Urteil. Jede Zeile lässt sich einzeln löschen, nach Bestätigung.</p>
<p><strong>Geprüft oder ungeprüft.</strong> Eine <em>geprüfte</em> Tabelle hat genau die Bytes, die gnubg für ihren Bereich erzeugt: ihr SHA-256-Fingerabdruck steht in blunderDB und wurde wiedergefunden. Die für die einseitigen Tabellen (OS-06 bis OS-10) hinterlegten Fingerabdrücke sind die, die das Werkzeug <code>makebearoff</code> von GNUbg 1.08 erzeugt. Eine <em>ungeprüfte</em> Tabelle ist wohlgeformt, aber für ihren Bereich ist kein Fingerabdruck hinterlegt — ihr wird nichts vorgeworfen, es hat sie nur niemand mit der Referenz verglichen. Eine <em>beschädigte</em> Tabelle widerspricht sich selbst und wird nie gelesen; sie wird neu berechnet.</p>
<p><strong>Eine breitere Tabelle berechnen.</strong> Der Bereich wird aus einer Liste zweier Familien gewählt, zusammen mit der Zahl der Kerne (standardmäßig alle bis auf einen, damit die Maschine benutzbar bleibt):</p>
<ul>
<li><strong>exaktes Doppler-Urteil (zweiseitig)</strong>, von TS-06-06 bis TS-06-15: erweitert den Bereich, in dem Gewinnwahrscheinlichkeit und Doppler-Urteil gelesen statt geschätzt werden;</li>
<li><strong>EPC außerhalb des Heimfelds (einseitig)</strong>, von OS-06 bis OS-10: erweitert, wie weit von zu Hause ein Stein stehen darf, ohne dass der EPC-Block verstummt. Dieser Durchlauf liest nur Stellungen, die kleiner sind als die berechnete, ist also von Natur aus sequentiell, und die Kernzahl bringt ihm nichts — der Auswähler sagt es, indem er ausgraut.</li>
</ul>
<p>Bevor irgendetwas beginnt, nennt der Reiter drei Zahlen für den gewählten Bereich: die Größe auf der Platte, den Speicher während der Berechnung und die Dauer <em>auf dieser Maschine</em>. Letztere beginnt als Schätzung und wird zur Messung: jeder ausreichend breite Lauf erfasst seine eigene Geschwindigkeit und behält sie. Ein Bereich, den der verfügbare Speicher nicht zulässt, wird ausgegraut angeboten, mit der Begründung — „nötig wären 24 GB, verfügbar sind 12“ ist eine Antwort, eine fehlende Zeile wäre keine.</p>
<p>Als Größenordnung, auf einer Maschine mit sechzehn Threads: TS-06-09 wiegt 191 MB und dauert etwa zehn Sekunden, TS-06-11 wiegt 1,2 GB und einige Minuten, TS-06-13 übersteigt, was die meisten Maschinen im Speicher halten können. Auf der einseitigen Seite, auf einem Kern: OS-07 wiegt 4,9 MB und dauert 17 s, OS-08 15 MB und 1 min 20, OS-10 117 MB und eine halbe Stunde.</p>
<p><strong>Pause und Fortsetzung.</strong> Während der Berechnung zeigt der Fortschritt die <em>gemessene</em> Restzeit und zwei getrennte Schaltflächen: <em>Pause</em> und <em>Abbrechen</em>. Die Pause schreibt den Zustand der Berechnung neben die Tabelle; ein erneuter Start setzt dort fort, statt von vorn zu beginnen. Abbrechen behält nichts. Das Schließen des Konfigurationsfensters unterbricht nichts — die Berechnung läuft im Hintergrund weiter.</p>
<p>Eine pausierte Berechnung findet sich beim nächsten Start wieder, benannt und beziffert („TS-06-09 bei 43 % unterbrochen“), mit <em>Fortsetzen</em> und <em>Löschen</em>. Nichts startet von selbst neu: der Benutzer hat um den Halt gebeten.</p>
<p>Der Reiter erlaubt schließlich, auf eine externe zweiseitige <code>.bd</code>-Datei zu zeigen, etwa eine von gnubg selbst erzeugte Datenbank: die Tabelle mit dem breitesten Bereich gewinnt.</p>
<p>Der Reiter <em>Allgemein</em> trägt schließlich <strong>Analysen reparieren</strong>: die Analysespalten, die Suche und Statistik abfragen, sind eine Projektion der gespeicherten Analysen, die unversehrt bleiben. Ein Fehler in der Projektion ist daher ohne erneuten Import reparierbar. Es ist ausdrücklich und nie automatisch — jemandes Analysespalten allein deshalb neu zu schreiben, weil er seine Datenbank öffnet, ist nichts, was ein Werkzeug hinter seinem Rücken tun sollte. Dasselbe <code>blunderdb repair</code> gibt es auf der Kommandozeile.</p>
<p>Der Reiter <strong>gammonNet</strong> stellt den eingebetteten Evaluator ein (siehe <code>ADR-0011 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0011-gammonnet-is-ported-to-go-and-the-representation-boundary-sits-at-the-evaluator-s-edge.md&gt;</code>__). Zwei Suchtiefen sind dort einstellbar, getrennt benannt und getrennt gespeichert — die eine abzusenken ändert nie die andere:</p>
<ul>
<li><strong>Anzeigetiefe</strong> — der interaktive Komfort während der Bearbeitung des Bretts; wird nie in die Datenbank geschrieben.</li>
<li><strong>Analysetiefe</strong> — das, was der Analyselauf nach dem Import in die Analyse einer Stellung schreibt.</li>
</ul>
<p>Beide stehen standardmäßig auf <strong>2-Ply</strong>, der kanonischen Konfiguration. Der Reiter bietet außerdem die <strong>Beschneidung</strong> (standardmäßig <code>k=12</code>) und die <strong>Anzahl der angezeigten Kandidatenzüge</strong> (standardmäßig 10) sowie ein Kontrollkästchen <strong>nach dem Import automatisch analysieren</strong>, das, einmal aktiviert, nach jedem Import prüft, ob Stellungen <strong>ohne jede Analyse</strong> übrig sind (weder gammonNet noch XG, GNUbg oder BGBlitz — die Regel lautet „eine Bewertung füllt nur eine Lücke“, nie ein Ersatz), und gegebenenfalls im Hintergrund eine gammonNet-Analyse in der konfigurierten Analysetiefe startet. Eine Schaltfläche <strong>Jetzt analysieren</strong> startet dieselbe Nachholanalyse manuell erneut — nützlich für eine Bibliothek, die vor dieser Funktion angelegt wurde.</p>
<p>Eine zweite Schaltfläche, <strong>Veraltete Positionen erneut analysieren</strong>, deckt den umgekehrten Fall ab: Eine bereits von gammonNet analysierte Stellung, deren gespeicherte Analyse jedoch mit einer älteren Engine-Version als der gerade laufenden oder mit einer anderen Tiefe als der oben eingestellten Analysetiefe geschrieben wurde, wird dort als veraltet markiert und neu bewertet. Eine Stellung, die zusätzlich eine XG-, GNUbg- oder BGBlitz-Analyse trägt, wird von dieser Schaltfläche nie angerührt, unabhängig von ihrem gammonNet-Inhalt — der Schutz von <code>ADR-0013 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0013-evaluations-fill-gaps-an-imported-analysis-is-never-overwritten.md&gt;</code>__ bleibt bedingungslos. Die neben jeder Schaltfläche angezeigte Anzahl (Stellungen ohne Analyse, veraltete Stellungen) ist rein informativ; der Stapel berechnet seine eigene Liste beim Start neu.</p>
<p>Beide Stapel sind <strong>begrenzt, sichtbar und abbrechbar, nie ein stiller Daemon</strong>: Ihr Fortschritt (<code>analysierte Stellungen / gesamt</code>) und eine Abbrechen-Schaltfläche erscheinen während ihrer gesamten Dauer in der Statusleiste und verschwinden nach Abschluss zugunsten einer Meldung, die das Ergebnis zusammenfasst — wie viele Stellungen <strong>analysiert</strong>, wie viele <strong>abgelehnt</strong> (eine Stellung, die gammonNet zu bewerten ablehnt, etwa ein Spielstand außerhalb der Reichweite seiner Tabelle, was nie ein Fehler ist) und wie viele <strong>fehlgeschlagen</strong> sind (beim nächsten Lauf unverändert erneut versucht). Das Schließen der Anwendung während des einen oder anderen verliert nichts: Jede analysierte Stellung wird laufend geschrieben, und der nächste Lauf setzt genau dort fort, wo die Analyse stehen geblieben war, ohne dass ein Journal zu führen wäre.</p>
<p><strong>Ein ohne Analyse importiertes Match erhält damit ein Performance Rating.</strong> Das betrifft ein online gespieltes Match oder eine Jellyfish-<code>.mat</code>-Datei, die niemand durch XG geschickt hat: blunderDB kannte die Stellungen und die gespielten Züge, aber keine Analyse sagte, was sie wert waren. Nach dem Lauf wird der tatsächlich gespielte Zug mit gammonNets Rangfolge verglichen, und die Differenz fließt in das PR, die Fehlerquote, die schlimmsten Entscheidungen und alle übrigen Kennzahlen ein, genau wie bei einem von XG analysierten Match. Der Vergleich erfindet nichts: der gespielte Zug stammt aus der Zugtabelle des Matches, beim Import geschrieben, ob die Datei nun eine Analyse trug oder nicht.</p>
<p>Eine mit einer älteren Version analysierte Datenbank muss nicht neu bewertet werden: <code>blunderdb repair</code> berechnet die Spalten aus den bereits gespeicherten Analysen und Zügen neu und gibt diesen Matches ihr PR zurück (siehe repair).</p>
<p>Ein ehrlicher Vorbehalt: eine Stellung wird durch ihre Struktur identifiziert, also trägt eine zweimal angetroffene Stellung — einmal gut, einmal schlecht gespielt — nur eine Differenz, die ihres ersten aufgezeichneten Vorkommens. Das ist nicht dieser Berechnung eigen: eine XG-Bibliothek hat genau dieselbe Form.</p>
<h4>Überwachter Ordner</h4>
<p>Der Reiter <strong>Überwachter Ordner</strong> weist blunderDB an, während des Betriebs einen Ordner zu beobachten und jede Matchdatei zu importieren, die darin <strong>erscheint</strong>. Eine Sitzung in eXtreme Gammon spielen, zu blunderDB zurückkehren, und die Matches sind schon da.</p>
<p>Nichts wird geraten. Solange kein Ordner benannt ist, gibt es keine Überwachung: blunderDB beginnt nicht, ein Verzeichnis zu lesen, weil es vermutet hat, wo Ihre Matches liegen. Die Schaltfläche <strong>Vorschlagen</strong> sieht an den üblichen Stellen dieses Rechners nach und schlägt eine nur vor, wenn sie wirklich existiert; sonst sagt sie es, und den Ordner zu benennen ist Ihre Sache.</p>
<p>Drei Dinge sollte man wissen, bevor man das Kästchen aktiviert:</p>
<ul>
<li><strong>Nur erscheinende Dateien werden importiert.</strong> Was der Ordner beim Start der Überwachung bereits enthält, wird als bekannt vermerkt und in Ruhe gelassen: eine Überwachung auf vier Jahre Matches zu richten darf nicht alle importieren. Um das Vorhandene zu importieren, gibt es den Ordnerimport — und beide ergänzen sich bestens, erst der Import, dann die Überwachung.</li>
<li><strong>Eine Datei wird erst importiert, wenn ihre Größe sich gesetzt hat.</strong> Ein Match, das ein anderes Programm gerade schreibt, wächst von einem Blick zum nächsten; es halb geschrieben zu importieren ergäbe einen Parserfehler, mit dem niemand etwas anfangen kann. blunderDB wartet daher, bis es dieselbe Datei zweimal unverändert gesehen hat.</li>
<li><strong>Der Import ist still.</strong> Sie haben gerade eine Stellung studiert, als Ihre Matches ankamen: Ihnen den Bildschirm wegzunehmen wäre der denkbar schlechteste Moment. Der Import läuft ohne Fenster, und die Statusleiste zeigt einen Streifen mit der Zahl der importierten, übersprungenen (Duplikate) und fehlgeschlagenen Matches, mit einer Schaltfläche, die auf Wunsch den vollständigen Bericht öffnet. Alles Übrige ist identisch mit einem manuellen Import: dieselbe Duplikaterkennung, derselbe Importlauf, dieselbe automatische Analyse, wenn sie eingeschaltet ist.</li>
</ul>
<p>Das Standardintervall beträgt zehn Sekunden; die Untergrenze zwei. Der Ordner wird nicht rekursiv durchlaufen: ein überwachter Ordner ist der Ort, an dem ein Werkzeug seine Matches ablegt, kein Baum zum Durchsuchen. Eine ausgehängte Netzwerkfreigabe beendet die Überwachung nicht und lässt ihren Inhalt bei der Rückkehr auch nicht als neu erscheinen.</p>
<p>Dieselbe Überwachung gibt es auf der Kommandozeile, mit <code>blunderdb import --type batch --dir &lt;Ordner&gt; --watch</code> (siehe Befehlszeilenschnittstelle (CLI)): es ist die Form, die ein Server, eine geplante Aufgabe oder ein Skript verwenden kann.</p>
<p>Das Konfigurationsfenster enthält außerdem Anzeigeeinstellungen für die Oberfläche. Ein Schieberegler für die <strong>Oberflächenskalierung</strong> ermöglicht es, alle Oberflächenelemente zu vergrößern oder zu verkleinern, was auf hochauflösenden Bildschirmen oder zur Verbesserung der Lesbarkeit nützlich ist. Ein Menü <strong>Panel-Position</strong> legt fest, wo die Panels (Suche, Matchs, Analyse) relativ zum Brett angezeigt werden: <em>unten</em>, <em>seitlich</em> oder <em>automatisch</em> (auf breiten Bildschirmen wird dann die seitliche Anordnung gewählt, um den verfügbaren Platz besser auszunutzen). Wie die anderen Einstellungen werden auch diese Festlegungen von einer Sitzung zur nächsten beibehalten.</p>
<h3>Geführte Touren und Beispieldatenbank</h3>
<p>Um den Einstieg zu erleichtern, bietet blunderDB <strong>geführte Touren</strong> durch die Oberfläche an. Der Katalog der Touren öffnet sich über die Symbolleiste oder mit dem Befehl <code>tour</code> (Alias <code>tutorial</code>). Es stehen sieben Touren zur Verfügung: eine allgemeine Tour durch die Oberfläche sowie Touren zur Stellungssuche, zur Durchsicht der Matches, zur Durchsicht der Turniere, zum Eval-Panel, zur Anki-Wiederholung und zu den Statistiken. Jede Tour hebt Schritt für Schritt die betreffenden Elemente der Oberfläche hervor, öffnet dabei das Panel, von dem sie spricht, und kann jederzeit wiederholt werden. Beim ersten Start wird die allgemeine Tour automatisch angeboten.</p>
<p>Der Befehl <code>demo</code> lädt eine <strong>Beispieldatenbank</strong>, mit der sich die Funktionen des Werkzeugs erkunden lassen, ohne eigene Partien zu importieren: drei Matches (zwei davon in einem Turnier zusammengefasst), analysiert von eXtreme Gammon, BGBlitz und gammonNet, drei thematische Sammlungen, mit Tags versehene Kommentare (<code>#blunder</code>, <code>#cube</code>) und ein Anki-Stapel mit seinem Wiederholungsprotokoll. Spieler, Turnier und Ort sind fiktiv. Die geführten Touren stützen sich auf diese Datenbank, wenn keine Datenbank geöffnet ist.</p>
<h3>Navigation durch die Stellungen</h3>
<p>Standardmäßig ermöglicht blunderDB:</p>
<ul>
<li>die verschiedenen Stellungen der aktuellen Bibliothek zu durchblättern — die niemals als Ganzes geladen wird: blunderDB führt nur die Liste der Kennungen und lädt die Stellungen in Fenstern von fünfzig um die gerade angezeigte herum, sodass eine Datenbank mit mehreren Zehntausend Stellungen genauso schnell öffnet wie eine kleine,</li>
<li>die mit einer Stellung verknüpften Analyseinformationen anzuzeigen,</li>
<li>die Kommentare einer Stellung anzuzeigen, hinzuzufügen und zu bearbeiten.</li>
</ul>
<p>Die Schaltfläche <strong>Zur Stellung springen</strong> in der Werkzeugleiste öffnet ein Fenster, in dem der Index einer Stellung direkt eingegeben werden kann, um dorthin zu springen, ohne zu blättern. Sie ist das grafische Gegenstück zum Befehl <code>[number]</code> in der Befehlszeile (siehe Positionen und Navigation).</p>
<div class="admonition tip">
<p>Siehe Tastenkürzel für die verfügbaren Tastenkürzel.</p>
</div>
<h3>Stellungen bearbeiten</h3>
<p>Das Drücken der Taste <em>TAB</em> öffnet das Suchpanel und ermöglicht es, eine Stellung auf dem Brett zu bearbeiten, um sie der Datenbank hinzuzufügen oder eine zu suchende Stellungsstruktur zu definieren. Die Verteilung der Steine, des Dopplers, des Punktestands und des Zugrechts können mit der Maus geändert werden (siehe Eine Position bearbeiten).</p>
<div class="admonition tip">
<p>Siehe Tastenkürzel für die verfügbaren Tastenkürzel.</p>
</div>
<h3>Die Befehlszeile</h3>
<p>Die in die Statusleiste integrierte Befehlszeile ermöglicht es, alle in der grafischen Oberfläche verfügbaren Funktionen von blunderDB auszuführen: allgemeine Operationen auf der Datenbank, Stellungsnavigation, Anzeige der Analyse und/oder Kommentare, Suche nach Stellungen mittels Filtern ... Nach einer ersten Einarbeitung in die Oberfläche empfiehlt es sich, allmählich die Befehlszeile zu verwenden, die eine leistungsfähige und flüssige Nutzung von blunderDB ermöglicht, insbesondere für die Funktionen zur Stellungssuche.</p>
<p>Um die Befehlszeile zu öffnen, drücken Sie die Taste <em>LEERTASTE</em>. Um eine Abfrage abzuschicken und die Befehlszeile zu schließen, drücken Sie die Taste <em>EINGABE</em>.</p>
<p>blunderDB führt die vom Benutzer gesendeten Abfragen aus, sofern sie gültig sind, und ändert gegebenenfalls sofort den Zustand der Datenbank. Es sind keine expliziten Speichervorgänge seitens des Benutzers erforderlich.</p>
<div class="admonition tip">
<p>Siehe Liste der Befehle für die Liste der in der Befehlszeile verfügbaren Befehle.</p>
</div>
<h3>Analyse-Panel</h3>
<p>Das Panel <strong>Analyse</strong> (<em>CTRL-L</em>) zeigt die Analysedaten der aktuellen Stellung an, importiert aus eXtreme Gammon (XG), GNUbg oder BGBlitz. Es stellt die besten Alternativen (Steinzüge oder Doppler-Entscheidungen) mit ihren Equity-Werten und den entsprechenden Fehlern dar. Die Taste <em>d</em> schaltet zwischen der Analyse der Steinzüge und der Analyse des Dopplers um. Beim Navigieren in einem Match wird der tatsächlich gespielte Zug in der Liste der Alternativen hervorgehoben. Drücken Sie <em>CTRL-L</em> oder führen Sie den Befehl <code>list</code> aus, um das Panel ein- oder auszublenden.</p>
<p>Unter den Tabellen sagt manchmal ein <strong>Satz</strong>, was die gespielte Entscheidung gekostet hat und warum: „Sie verlieren 120 mMWC: Der gespielte Zug lässt drei Blots stehen, 13/7 8/7 nur einen.“ Er stammt aus sechs messbaren Regeln — Blößen, ein gemachter oder verpasster Heimfeldpunkt, aufgegebene Gammon-Chancen, eine Sicherheit, die mehr kostet als sie bringt, und die beiden Richtungen eines Verdopplungsfehlers (zu spät oder zu früh doppeln, zu locker annehmen oder zu eng aufgeben).</p>
<p>Die Regel, auf die es ankommt, ist das <strong>Schweigen</strong>: Der Satz erscheint nur, wenn eine Regel sicher greift, und bei einem Fehler jenseits der Schwelle, ab der die Engines übereinstimmen, dass es einer ist. Sonst gibt es keinen Satz — keinen leeren Rahmen, kein „wir wissen es nicht“. Eine falsche Erklärung kostet mehr als keine: Sie lehrt etwas Unzutreffendes.</p>
<p>Wurde eine Stellung von <strong>mehreren Engines</strong> beurteilt, stellt ein Streifen am Kopf des Panels sie nebeneinander: eine Zeile je Engine, mit ihrer Tiefe und ihrer Antwort — dem Würfelurteil oder ihrem eigenen besten Zug. Er sagt zuerst, ob sie übereinstimmen, und der Widerspruch ist es, der ihn rechtfertigt: „XG sagt Doppel, Annahme; gammonNet sagt kein Doppel“ liest sich auf einen Blick, wo zuvor zwei Tabellen quer verglichen werden mussten.</p>
<p>Der beste Zug einer Engine ist der beste <strong>dieser Engine</strong>: die Kandidatenliste ist über alle Engines hinweg nach Equity sortiert, ihr erster Eintrag ist also niemandes bester Zug im Besonderen.</p>
<p>Der Streifen erscheint nur, wenn es tatsächlich mehrere Engines gibt, und es gibt ihn allein in diesem Panel: das Eval-Panel zeigt <strong>eine</strong> Entscheidung, die der eingebetteten Engine (<code>ADR-0017 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0017-the-panel-shows-position-facts-plus-the-one-decision-the-board-asks.md&gt;</code>__), und ein Vergleich hätte dort keinen Platz.</p>
<p>Züge werden so geschrieben, wie man sie auf dem Brett liest, hier wie im Eval-Panel: der am wenigsten vorgerückte Stein zieht zuerst, und <strong>ein Stein, der mehrere Würfel hintereinander nutzt, wird nur einmal geschrieben</strong> — eine mit demselben Stein gespielte 64 liest sich <code>24/14</code>, und <code>24/14*</code>, wenn er bei der Ankunft schlägt. Das Detail der Kette taucht nur wieder auf, wenn es etwas mehr aussagt: ein Schlag <em>unterwegs</em> behält seinen Zwischenpunkt, <code>24/18* 18/14</code>, sonst verschwände der Schlag auf der 18 aus der Notation.</p>
<p>Die Equity einer importierten Analyse folgt derselben Regel wie das Eval-Panel: Die Spalte nennt ihren Bezugsrahmen, „Equity (money)“ oder „Equity (match)“ je nach Spielstand der analysierten Position, nie ein bloßes „Equity“ ohne Angabe der Skala. Die auf einer Money-Game-Position aktiven Regeln <strong>Jacoby</strong> und <strong>Beaver</strong> werden ebenfalls angezeigt, als Badges unter der Tabelle der Doppelwürfel-Entscheidung.</p>
<h3>Kommentare-Panel</h3>
<p>Das Panel <strong>Kommentare</strong> (<em>STRG-P</em>) zeigt, ergänzt und bearbeitet die Kommentare zur aktuellen Stellung. Eine Stellung kann mehrere tragen: alle werden angezeigt, die neuesten zuerst. Aus XG-Dateien importierte Kommentare werden den passenden Stellungen automatisch zugeordnet. <em>STRG-P</em> drücken oder den Befehl <code>comment</code> ausführen, um das Panel ein- oder auszublenden.</p>
<p>Jeder Kommentar aus einer Datei trägt eine <strong>Herkunftsmarkierung</strong> (<code>XG</code>, <code>GNU BG</code>, <code>BGF</code>, oder <em>importiert</em>, wenn die Herkunft nie festgehalten wurde). Von Ihnen geschriebene Kommentare tragen keine: das ist der Normalfall, und jede Zeile zu kennzeichnen wäre nur Lärm. Einen importierten Kommentar zu bearbeiten macht ihn zu Ihrem: nach der Änderung ist der Satz Ihrer.</p>
<p>Diese Unterscheidung wirkt sich anderswo aus: das Löschen einer Partie zerstört keine Stellung mehr, auf die <strong>Sie</strong> etwas geschrieben hatten. Eine aus der Quelldatei übernommene Notiz verschwindet dagegen weiterhin mit der Partie, die sie mitgebracht hat.</p>
<h4>Tags</h4>
<p>Ein <strong>Tag</strong> ist ein <code>#Wort</code> in einem Kommentar. Nichts deklariert es, keine Tabelle hält es, und das ist Absicht: das Vokabular ist Ihre eigene Prosa, und eine Deklaration zu verlangen, bevor man taggen darf, machte aus einer Gewohnheit Papierkram.</p>
<p>Was fehlte, war die andere Hälfte: das Vokabular, das man sich aufgebaut hat, zu <strong>sehen</strong> und ein Tag anzuklicken, statt sich zu erinnern, wie man es geschrieben hat. Der Befehl <code>tags</code> oder die Schaltfläche <code>#</code> neben dem Eingabefeld öffnet das Vokabularfenster: die Tags dieser Datenbank, jedes mit der <strong>Zahl der Stellungen</strong>, die es tragen, anklickbar, um die entsprechende Suche zu starten. Unter der Liste stehen die empfohlenen Tags, die diese Datenbank noch nicht verwendet — ein Vokabular aus der Backgammon-Literatur (<code>#blitz</code>, <code>#prime</code>, <code>#holding</code>, <code>#backgame</code>, <code>#containment</code>, <code>#crunch</code>, <code>#ace-point</code>, <code>#timing</code>…), vorgeschlagen und nie vorgeschrieben: ein Tag, das nicht auf dieser Liste steht, ist genau so viel wert wie eines, das darauf steht.</p>
<p>Beim Tippen bietet ein <code>#</code> die Tags an, die <strong>diese Datenbank</strong> bereits verwendet, dann die empfohlenen. Das verhindert, dass man an einem Tag <code>#back-game</code> und am nächsten <code>#backgame</code> schreibt — was sonst nichts auffangen würde.</p>
<p>Eine Tag-Suche schreibt man auf der Kommandozeile als <code>#prime</code>. Sie ist <strong>abgegrenzt</strong>: <code>#prime</code> findet <code>#priming</code> nicht, während eine gewöhnliche Textsuche, die eine Teilzeichenkette sucht, die beiden nicht unterscheiden kann. Mehrere Tags <strong>addieren sich</strong> — <code>s #prime #backgame</code> verlangt die Stellungen, die beide tragen — denn eine Stellung trägt mehrere Tags: zwei zu nennen kann nur „beide“ heißen. Das ist das Gegenteil des Phasen- oder Herkunftsfilters, wo eine Stellung nur einen Wert hat und zwei Werte zu nennen nur „das eine oder das andere“ heißen kann.</p>
<p>Dieselbe Liste erhält man außerhalb der Oberfläche mit <code>blunderdb list --type tags</code> (siehe Befehlszeilenschnittstelle (CLI)).</p>
<h3>Der Papierkorb</h3>
<p>Eine Stellung, eine Sammlung oder einen Kommentar zu löschen geht jetzt über einen <strong>Papierkorb</strong>: der Löschvorgang findet statt, aber eine Kopie dessen, was verschwindet, wird dreißig Tage aufbewahrt. Der Befehl <code>trash</code> öffnet das Fenster, das sie auflistet, jede mit <em>Wiederherstellen</em> und <em>Löschen</em>.</p>
<p>Eine wiederhergestellte Stellung kommt <strong>mit ihrer Analyse und ihren Kommentaren</strong> zurück — sie nackt zurückzugeben wäre eine Wiederherstellung nur dem Namen nach. Sie kommt nicht unter ihrer alten Nummer zurück: die ursprüngliche Zeile existiert nicht mehr, und blunderDB speichert sie über ihren Fingerabdruck neu, was garantiert, dass nie ein Duplikat entsteht, ihr aber eine neue Kennung gibt. Eine Sammlung kommt mit ihrer Liste zurück; die Stellungen, die sie enthielt, waren nie gelöscht — eine Sammlung ist eine Sicht auf sie.</p>
<p>Was älter als dreißig Tage ist, entfernt der Befehl <code>vacuum</code>, nie das Öffnen einer Datenbank: kein <code>vacuum</code> heißt, alles zu behalten.</p>
<div class="admonition note">
<p>Der Papierkorb reist nicht mit. Ein Export trägt ihn nicht, und das Löschen einer Partie legt nichts hinein: die Waisen-Bereinigung nach einer Partielöschung ist automatische Hausarbeit, keine Geste des Benutzers — siehe die Aufbewahrungsregel in Matches-Panel.</p>
</div>
<h3>Such-Panel</h3>
<p>Das Panel <strong>Suche</strong> (<em>CTRL-F</em> oder <em>TAB</em>) ermöglicht es, Stellungen nach frei kombinierbaren Kriterien zu filtern: Steinstruktur, Typ der Doppler-Entscheidung, Fehlergröße, Datum, Tags usw. Die Taste <em>TAB</em> öffnet gleichzeitig das Suchpanel und den Stellungseditor, sodass eine zu suchende Steinstruktur direkt auf dem Brett definiert werden kann.</p>
<p>Um eine Suche unter den aktuell gefilterten Stellungen zu verfeinern, verwenden Sie den Befehl <code>ss</code> gefolgt von Filtern (z. B.: <code>ss nc</code>, <code>ss E&gt;40</code>). Das Suchpanel bietet für dieselbe Funktion auch ein Kontrollkästchen <em>In aktuellen Ergebnissen suchen</em>.</p>
<p>Das Panel bietet eine explizite Steuerung des gesuchten <strong>Entscheidungstyps</strong>: <em>Egal</em> (kein Filter), <em>Zug</em> (Zugentscheidungen) oder <em>Dopplung</em> (Doppler-Entscheidungen). Wenn <em>Dopplung</em> ausgewählt ist, gibt eine zweite Liste den Untertyp an: <em>Alle</em>, <em>Dopplung / Kein Doppel</em> (der Spieler am Zug muss über das Doppeln entscheiden) oder <em>Annehmen / Aufgeben</em> (Antwort auf ein gegnerisches Doppel). Die Steuerung ist mit dem Brett synchronisiert: Ändert man die Würfel oder den Dopplerwürfel auf dem Brett, wird der Entscheidungstyp aktualisiert und umgekehrt. Im Modus <em>Annehmen / Aufgeben</em> wird der Dopplerwürfel mit dem angebotenen Wert in der Mitte des Bretts angezeigt; dieser Wert bleibt bearbeitbar.</p>
<p>Die <strong>Spielphase</strong> — Eröffnung, Mittelspiel, Wettlauf, Auswürfeln — ist eine Kennzeichnung, die blunderDB allein aus dem Brett berechnet. Sie ist nie editierbar und über das Kommandozeilen-Token <code>ph:</code> durchsuchbar (<code>ph:race</code>, wiederholbar: <code>ph:race ph:bearoff</code>). Drei ihrer vier Grenzen sind die, mit denen GNU Backgammon seine Netze auswählt; die vierte, wo die Eröffnung endet, ist eine Konvention von blunderDB: eine Stellung befindet sich noch in der Eröffnung, solange keine Seite mehr als vier Steine von ihren Ausgangspunkten bewegt hat, nichts ausgewürfelt wurde und nichts auf der Bar steht.</p>
<div class="admonition note">
<p>Die Kennzeichnung wird vom Befehl <code>blunderdb repair</code> neu berechnet. Bei einer Datenbank, die zum ersten Mal mit dieser Version geöffnet wird, geschieht das einmal beim Öffnen. Eine Datenbank, deren Phasen nie berechnet wurden, liefert für <code>ph:</code> nichts — nichts statt einer falschen Antwort.</p>
</div>
<p>Der Befehl <code>like</code> beantwortet eine andere Frage als die Token: Er ersetzt die durchblätterte Liste durch die Stellungen, die der aktuellen am <strong>nächsten</strong> stehen, die nächste zuerst. Die Nähe ist eine Transportdistanz in Stein-Pips — die Menge an Steinbewegung, die die beiden Stellungen trennt — und der Blickwinkel ist immer der des Spielers am Zug. Es ist kein Filter: Die Ähnlichkeit <strong>ordnet</strong> die ganze Datenbank, statt sie einzuschränken, und lässt sich daher nicht mit den Token kombinieren.</p>
<p>Das Token <code>n</code> zählt <strong>Begegnungen</strong>: <code>n&gt;3</code> behält die Stellungen, die mehr als drei Züge erreichen, über alle Matches hinweg. Das ist eine andere Frage als „was habe ich falsch gemacht“ — eine Stellung, die zwanzigmal vorkam und neunzehnmal richtig gespielt wurde, ist immer noch die, die man auswendig können muss. Gezählt werden Züge, nicht Matches: dieselbe Stellung zweimal in einem Match zählt zweimal, denn das waren zwei Entscheidungen.</p>
<p>Ein Satz in Worten kann die Token ersetzen, mit dem Befehl <code>ask</code>: <code>ask my cube blunders at a score</code>. Der Satz wird <strong>in Token übersetzt</strong>, die in die Befehlszeile geschrieben werden — durchlesen, dann ausführen. Nichts wird erraten und nichts verlässt den Rechner: Das Vokabular ist fest, derselbe Satz ergibt immer dieselbe Abfrage, und was nicht verstanden wurde, wird <strong>gesagt</strong> statt übergangen. Eine falsche Übersetzung sieht man so, bevor sie falsche Ergebnisse liefert, und die Token lernt man beim Lesen.</p>
<p>Zwei Absichten sind keine Token und werden auf dem Suchbrett gesetzt statt in der Zeile: „Verdopplung“ oder „Zug“ (die Art der Entscheidung) und „bei Matchstand“ oder „Money“. <code>ask</code> setzt sie dort.</p>
<p>Der <strong>Spielplan</strong> ist ein zweites abgeleitetes Etikett neben der Phase, und es beantwortet die Frage, die ein Bündel gespeicherter Filter nicht stellen kann: „zeig mir meine Fehler im Holding Game“. Token <code>gt:</code>, wiederholbar (<code>gt:holding gt:mutualholding</code>), aus Sicht des <strong>Spielers am Zug</strong> — des Plans, in dem die Entscheidung fiel.</p>
<p>Die zehn erkannten Pläne, in der Reihenfolge, in der die Regeln sie abarbeiten, vom Spezifischsten zum Allgemeinsten:</p>
<ul>
<li><code>race</code> — die hintersten Steine beider Seiten haben sich gekreuzt: Kontakt ist nicht mehr möglich. Grenze von GNU Backgammon.</li>
<li><code>bearin</code> — der Spieler am Zug bringt ein, während der Gegner noch einen Anker in dessen Heimfeld hält.</li>
<li><code>crunch</code> — der Spieler am Zug hat höchstens sechs Steine außerhalb seiner Punkte 1 und 2. Regel von GNU Backgammon, Schwelle ihres Autors.</li>
<li><code>backgame</code> — zwei oder mehr Anker im gegnerischen Heimfeld.</li>
<li><code>acepoint</code> — ein einziger Anker, auf dem gegnerischen Ass-Punkt, mit mindestens zwanzig Pips Rückstand.</li>
<li><code>blitz</code> — drei oder mehr Heimfeldpunkte gemacht, und der Gegner auf der Bar oder mit einem Blot zum Schlagen in diesem Heimfeld.</li>
<li><code>primevprime</code> — beide Seiten halten eine Prime von mindestens vier Punkten, und jede hat einen Stein hinter der des anderen gefangen.</li>
<li><code>mutualholding</code> — beide Seiten halten einen hohen Anker.</li>
<li><code>holding</code> — der Spieler am Zug hält einen hohen Anker, der Gegner nicht.</li>
<li><code>contact</code> — Kontakt, und keiner der obigen Pläne. Die Eröffnung landet hier.</li>
</ul>
<p>Drei dieser Regeln sind die von GNU Backgammon und sind belegt; die übrigen sind <strong>Konventionen von blunderDB</strong>. Die Backgammon-Literatur beschreibt die Spielpläne, ohne ihre Grenzen zu beziffern, und für dieses Problem ist keine Übereinstimmung zwischen Klassifizierern veröffentlicht. Die nicht belegten Schwellen — drei Heimfeldpunkte für einen Blitz, vier Punkte für eine Prime, zwanzig Pips Rückstand für ein Ace-Point-Game — stehen deshalb hier statt versteckt im Code, und sie sind versioniert: ändern, <code>blunderdb repair</code> laufen lassen, und die ganze Datenbank ist neu etikettiert.</p>
<div class="admonition note">
<p>Pro Stellung wird ein Etikett behalten, das des Spielers am Zug. Ein abgeleitetes Etikett ist nie editierbar, wird nie als Wahrheit exportiert, und eine Datenbank, deren Pläne nie berechnet wurden, liefert für <code>gt:</code> nichts — wie für <code>ph:</code>.</p>
</div>
<p>Der Filter <strong>Markiert</strong> behält die Positionen, die Sie in der Ursprungssoftware der Partie markiert haben. Nur eXtreme Gammon erzeugt diese Information, die zugweise in der <code>.xg</code>-Datei gespeichert wird; blunderDB liest sie beim Import und behält sie bei. Eine markierte Dopplungsentscheidung ergibt zwei markierte Positionen, die Dopplung und das Annehmen/Aufgeben, da blunderDB in zwei Teile zerlegt, was die Quelldatei als eine einzige Entscheidung speichert.</p>
<div class="admonition note">
<p>Die Markierung wirkt nicht rückwirkend: Bereits in der Datenbank vorhandene Partien tragen diese Information nicht, da sie nur in den Quelldateien existiert. Importieren Sie einfach die betreffende <code>.xg</code>-Datei erneut — der Import erkennt das Duplikat und fügt nichts außer den Markierungen hinzu, ohne vorhandene Kommentare oder Analysen anzutasten. Eine Markierung kann in blunderDB weder gesetzt noch entfernt werden: Für eine temporäre Arbeitsliste verwenden Sie besser eine Sammlung.</p>
</div>
<p>Der Filter <strong>Kommentar</strong> durchsucht die den Positionen zugeordneten Kommentare in drei sich ausschließenden Modi. <em>enthält Text</em> sucht ein oder mehrere Wörter im Kommentartext (Eingabefeld, Wörter durch <code>;</code> getrennt, mindestens eines muss zutreffen); <em>hat einen Kommentar</em> behält jede Position mit einem Kommentar, unabhängig vom Inhalt; <em>ohne Kommentar</em> behält im Gegenteil die nicht kommentierten Positionen — nützlich in Kombination mit einem Fehler- oder Datumsfilter, um die Liste dessen zu erstellen, was noch zu kommentieren ist.</p>
<div class="admonition note">
<p>Aus einer Partiedatei (XG, GNUbg) importierte Kommentare zählen als Kommentare. Um nur Ihre eigenen zu behalten, fügen Sie auf der Kommandozeile das Token <code>co:user</code> hinzu (<code>co:xg</code>, <code>co:gnubg</code>, <code>co:bgf</code> und <code>co:unknown</code> benennen die anderen Herkünfte). Kommentare zu einer <em>Partie</em> oder einem <em>Turnier</em> sind ohnehin nicht betroffen: sie kommentieren die Partie oder das Turnier, nicht deren Stellungen.</p>
</div>
<p>Der Filter <strong>Matches &amp; Turniere</strong> stützt sich auf einen gemeinsamen Auswahldialog (modales Fenster) anstelle der Eingabe numerischer Kennungen: zwei Kontrollkästchen-Listen, eine für Matches und eine für Turniere, jeweils per Text filterbar (Spieler, Datum, Ereignis für Matches; Name, Datum, Ort für Turniere), mit den Schaltflächen <em>Alle</em> / <em>Keine</em>, die nur auf die aktuell gefilterte Teilmenge wirken. Das Ankreuzen eines Turniers kreuzt automatisch dessen zugehörige Matches in der Matchliste an (und graut sie aus), wodurch sichtbar wird, dass ein Turnier der Gesamtheit seiner Matches entspricht.</p>
<p>Das Suchpanel enthält an seinem linken Rand drei Reiter: <em>Suche</em> (die Filter), <em>Verlauf</em> und <em>Gespeichert</em>. Der Reiter <strong>Verlauf</strong> listet die vergangenen Suchen mit ihrem Datum und ihrem Befehl auf: Ein Klick wählt eine Suche aus und zeigt die zugehörige Position auf dem Brett an, ein Doppelklick führt sie erneut aus. Jeder Eintrag kann in der Filterbibliothek gespeichert (Lesezeichen-Symbol, durch Angabe eines Filternamens) oder gelöscht werden. Der Reiter <strong>Gespeichert</strong> enthält die <strong>Filterbibliothek</strong>: Doppelklicken Sie auf einen gespeicherten Filter, um die entsprechende Suche erneut zu starten (siehe Anhang: Fortgeschrittene Nutzung der Filter). Der Befehl <code>history</code> (Alias <code>hi</code>) öffnet das Suchpanel.</p>
<div class="admonition tip">
<p>Siehe Liste der Befehle für die Liste der verfügbaren Filter.</p>
</div>
<h3>Sammlungen-Panel</h3>
<p>Das Fenster <strong>Sammlungen</strong> (<em>CTRL-B</em>) verwaltet Stellungssammlungen. Sammlungen können angelegt, umbenannt und gelöscht werden. Stellungen können hinzugefügt oder entfernt werden (Taste <em>Entf</em>, Bestätigung wird verlangt). Ein Doppelklick auf eine Sammlung durchblättert ihre Stellungen mit den Tasten <em>LINKS</em> und <em>RECHTS</em>. Die Reihenfolge der Sammlungen und der Stellungen innerhalb einer Sammlung lässt sich per Ziehen und Ablegen ändern. <em>CTRL-B</em> drücken oder den Befehl <code>collection</code> ausführen, um das Fenster ein- oder auszublenden.</p>
<h3>Import: was geschrieben wird, was es niemals ist</h3>
<p>Das Importieren eines Matches, einer Stellung oder einer anderen Datenbank fügt hinzu, was fehlt; es ersetzt nicht, was bereits da ist.</p>
<ul>
<li><strong>Eine Stellung wird niemals dupliziert.</strong> Ihre Identität — Steine, Doppler, Würfel, Spielstand — erkennt sie, niemals die Datei, aus der sie stammt: Dieselbe in zwei Matches angetroffene Stellung bleibt eine einzige Zeile.</li>
<li><strong>Eine Analyse pro Engine.</strong> eXtreme Gammon, GNUbg, BGBlitz und der eingebaute Evaluator koexistieren auf derselben Stellung, und das Analyse-Panel gibt die Herkunft jeder einzelnen an. Das Importieren der einen löscht die andere nicht.</li>
<li><strong>Eine importierte Analyse wird niemals neu berechnet.</strong> blunderDB legt sie unverändert ab, mit ihrer Niveau-Kennzeichnung („3-ply“, „XG Roller++“, „Book“), ihren Equities, ihren Fehlern, ihren Wahrscheinlichkeiten und dem Würfelglück. Die Regel lautet „eine Bewertung füllt nur eine Lücke“: Die automatische Analyse nach dem Import besucht nur Stellungen ohne <strong>jegliche</strong> Analyse, und <em>Veraltete Positionen erneut analysieren</em> lässt jede Stellung mit einer importierten Analyse unangetastet (siehe Konfiguration).</li>
<li><strong>Dieselbe Datei erneut zu importieren schreibt nichts um.</strong> Das Match wird als bereits vorhanden erkannt; nur die in der Ursprungssoftware gesetzten Markierungen werden hinzugefügt, ohne die Kommentare oder Analysen zu berühren.</li>
<li><strong>Was blunderDB niemals schreibt</strong>: ein neu berechnetes Glück — es wird aus der Quelldatei gelesen oder bleibt unbekannt — und ein Rollout, dessen Daten es in einer <code>.xg</code>-Datei nicht öffnet und das es nicht erzeugen kann.</li>
</ul>
<p>Eine Sammlung kann <strong>lebendig</strong> sein: Ihr Inhalt ist keine handgemachte Liste mehr, sondern das Ergebnis einer <strong>Suche</strong>, bei jedem Öffnen neu ausgewertet. Die Schaltfläche ◇ am Kopf der Sammlung macht sie mit der zuletzt ausgeführten Suche lebendig; ◈ sagt, dass sie es schon ist, und dieselbe Schaltfläche gibt ihr die Liste zurück. Nichts wird zerstört: Die Stellungen, die sie enthielt, sind beim Zurückgehen noch da.</p>
<p>Eine lebendige Sammlung, deren Abfrage ein Token trägt, das diese Version nicht mehr kennt, <strong>weigert sich zu öffnen</strong> und sagt es, statt die ganze Datenbank zurückzugeben. Das ist der eine Fehler, den ein gespeicherter Filter nicht haben darf: sich im Stillen zu weiten.</p>
<h3>Matches-Panel</h3>
<p>Das Panel <strong>Matches</strong> (<em>CTRL-Tab</em>) listet die importierten Matches auf. Doppelklicken Sie auf ein Match (oder drücken Sie <em>EINGABE</em>), um durch seine Züge zu navigieren. Der Befehl <code>m</code> setzt die Navigation im zuletzt besuchten Match fort.</p>
<p>Der Benutzer kann:</p>
<ul>
<li>die Züge eines Matches mit den Tasten <em>LINKS</em> und <em>RECHTS</em> durchblättern,</li>
<li>mit den Tasten <em>PageUp</em> und <em>PageDown</em> von einer Partie zur anderen wechseln,</li>
<li>die Analyse der Züge (Steine und Doppler) durch Drücken von <em>CTRL-L</em> anzeigen,</li>
<li>mit der Taste <em>d</em> zwischen der Analyse der Steinzüge und des Dopplers umschalten,</li>
<li>den tatsächlich gespielten Zug in der Analyse hervorgehoben sehen.</li>
</ul>
<p>Die zuletzt besuchte Stellung in jedem Match wird gespeichert und automatisch wiederhergestellt. Drücken Sie <em>CTRL-Tab</em> oder führen Sie den Befehl <code>match</code> aus, um das Panel ein- oder auszublenden.</p>
<p>Die Schaltfläche <strong>⊕</strong> einer Zeile reichert dieses Match aus einer Datei an. Dahinter steckt nichts Neues: dasselbe Match in einem anderen Format erneut zu importieren reichert es bereits an Ort und Stelle an — der kanonische Hash erkennt, dass es dasselbe Match ist, und die Analysen und Kommentare der zweiten Datei ergänzen die erste. Was die Schaltfläche bringt, ist, dass man sie findet: niemand errät, dass ein Import auch eine Anreicherung ist. Der folgende Bericht sagt, welches von beiden geschehen ist — „angereichert: 1“ statt „importiert: 1“.</p>
<p>Jedes Match kann über die Schaltfläche ⬇ in der Matchliste oder die Schaltfläche <em>.mat</em> auf der Match-Karte als Jellyfish-Transkription <code>.mat</code> exportiert werden.</p>
<p>Die Schaltfläche <strong>Spieler zusammenführen</strong> in der Werkzeugleiste des Panels öffnet ein Fenster, das alle Spielernamen der Datenbank mit ihrer Anzahl an Matches auflistet: die Schreibvarianten desselben Spielers auswählen, den beizubehaltenden kanonischen Namen wählen und dann zusammenführen. Nützlich, um die Statistiken pro Spieler zu vereinheitlichen, wenn derselbe Spieler unter mehreren Namen erscheint.</p>
<p>Wenn ein Match geöffnet ist, erscheint über dem Brett eine <strong>Informationsleiste</strong>: Sie zeigt die beteiligten Spieler (<em>Spieler 1</em> gegen <em>Spieler 2</em>) sowie den Kontext des Matches (Ereignis, Ort, Runde, Datum und Matchlänge, sofern diese Informationen verfügbar sind). Diese Leiste wird auch außerhalb des Match-Modus angezeigt: Wenn eine untersuchte Position (aus einer Suche, einer Sammlung oder einem direkten Zugriff) aus einem oder mehreren Matches stammt, gibt sie deren <strong>Herkunft</strong> an — das erste betroffene Match und gegebenenfalls ein Badge „+N“, das die übrigen beim Überfahren auflistet. Eine einzeln importierte Position, auf die kein Match verweist, zeigt nichts an.</p>
<p>Beim Öffnen einer Datenbank, die Matchs enthält, wird das Panel <strong>Matchs</strong> sofort angezeigt und die Durchsicht beginnt direkt bei der ersten Stellung, sodass Sie unmittelbar mit der Navigation beginnen können.</p>
<div class="admonition note">
<p>Eine Datenbank kann jeweils nur von einem einzigen Fenster zum Schreiben geöffnet werden. Wenn Sie eine Datenbank öffnen, die bereits in einem anderen blunderDB-Fenster geöffnet ist, wird sie <strong>schreibgeschützt</strong> geöffnet: Navigation, Suche und Analyse bleiben möglich, aber jede Änderung ist deaktiviert und die Titelleiste zeigt „[schreibgeschützt]“ an.</p>
</div>
<div class="admonition tip">
<p>Siehe Tastenkürzel für die verfügbaren Tastenkürzel.</p>
</div>
<h3>Turnier-Panel</h3>
<p>Das Panel <strong>Turniere</strong> (<em>CTRL-Y</em>) ermöglicht es, Matches in Turnieren zu gruppieren, um eine organisierte Nachverfolgung und eine statistische Analyse pro Veranstaltung zu ermöglichen. Turniere können erstellt, umbenannt und gelöscht werden; Matches können ihnen zugeordnet werden. Die Statistiken des Stats-Panels können nach Turnier gefiltert werden. Drücken Sie <em>CTRL-Y</em>, um das Panel ein- oder auszublenden.</p>
<p>Turniere füllen sich beim Import von selbst. XG-, GnuBG- und BGF-Dateien nennen ihr Event; wird ein neues Match importiert, ordnet blunderDB es dem Turnier dieses Namens zu und legt dieses an, falls es noch nicht existiert. Datum und Ort des Turniers bleiben leer — hier werden sie eingetragen. Ein Match, das bereits in der Datenbank ist, wird nie umsortiert: seine Datei erneut zu importieren macht eine von Hand vorgenommene Einordnung nicht rückgängig.</p>
<p>Die Spalte <strong>PR</strong> jedes Turniers zeigt den PR des <strong>Referenzspielers</strong> — also des Spielers, der in den meisten Partien des Turniers vorkommt (bei Gleichstand derjenige mit den meisten Entscheidungen). Der PR vermischt Ihr Spiel also nicht mit dem Ihrer Gegner: Bei Ihren eigenen Turnieren spiegelt er allein Ihre Leistung wider. Der Name des Referenzspielers erscheint als Tooltip, wenn Sie den Wert überfahren.</p>
<h3>Stats-Panel</h3>
<h4>Einführung</h4>
<p>Das Panel <strong>Stats</strong> ermöglicht es, das eigene Spielniveau zu analysieren und den Fortschritt im Zeitverlauf anhand der in die Datenbank importierten Stellungen zu verfolgen. Es berechnet und zeigt die Kennzahlen <strong>PR</strong> (Performance Rating) und <strong>MWC cost</strong> (Match Winning Chance cost) für alle Stellungen oder eine gefilterte Teilmenge an.</p>
<p>Das Stats-Panel ist besonders nützlich, um:</p>
<ul>
<li><strong>das eigene Niveau einzuordnen</strong> im Verhältnis zu den Niveaubändern (<em>Weltklasse</em>, <em>Experte</em>, *Fortgeschritten*…) anhand des globalen PR;</li>
<li><strong>den eigenen Fortschritt zu verfolgen</strong> Turnier für Turnier oder Match für Match anhand der Diagramme im Tab Progression;</li>
<li><strong>die eigenen Schwachstellen zu erkennen</strong>: der Tab Fehler zeigt die Aufteilung zwischen Steinzügen und Doppler-Entscheidungen sowie die Verteilung der Fehlergrößen;</li>
<li><strong>die Spieler der Datenbank untereinander vergleichen</strong>, eine Zeile je Spieler, über den Reiter Spieler — nützlich, um ein ganzes Turnier zu verfolgen;</li>
<li><strong>direkt zu den betreffenden Stellungen zu gelangen</strong>, indem man auf eine beliebige Kennzahl klickt (Drill-down).</li>
</ul>
<h4>Öffnen des Panels</h4>
<p>Um das Stats-Panel zu öffnen:</p>
<ul>
<li>Drücken Sie <em>CTRL-D</em>.</li>
<li>Geben Sie den Befehl <code>stats</code> oder <code>st</code> in die Befehlszeile ein.</li>
</ul>
<div class="admonition note">
<p>Das Panel wird bei jeder Änderung des Filters automatisch aktualisiert. Bei einem einfachen Umschalten PR ↔ MWC werden die Statistiken nicht neu berechnet: Beide Kennzahlen werden vom Backend gleichzeitig berechnet.</p>
</div>
<h4>Filterleiste</h4>
<p>Die Filterleiste am oberen Rand des Panels ermöglicht es, die Berechnung auf eine Teilmenge von Stellungen einzuschränken.</p>
<h5>Spielerperspektive</h5>
<p>Die Dropdown-Liste <strong>Spieler</strong> ermöglicht es, die Statistiken nach dem analysierten Spieler zu filtern. blunderDB wählt automatisch den Spieler aus, dessen Name am häufigsten in der Datenbank vorkommt — jederzeit änderbar.</p>
<div class="admonition tip">
<p>Ein Spielerwechsel führt nicht zu Datenverlust; es genügt, den vorherigen Spieler in der Liste erneut auszuwählen.</p>
</div>
<h5>Verfügbare Filter</h5>
<ul>
<li><strong>Turnier(e)</strong> — Beschränkung auf ein oder mehrere Turniere. Es können mehrere Turniere gleichzeitig ausgewählt werden.</li>
<li><strong>Datum</strong> — Zeitspanne (<em>Von</em> … <em>Bis</em>). Wenn nur das Startdatum angegeben ist, werden die neueren Stellungen einbezogen.</li>
<li><strong>Entscheidungstyp</strong> — Alle / Steinzüge / Doppler-Entscheidungen.</li>
<li><strong>Matchlänge</strong> — Beschränkung auf bestimmte Matchlängen (1, 3, 5, 7, 9, 11, 13, 15, 21 Punkte). Es können mehrere Längen kombiniert werden.</li>
</ul>
<p>Eine Schaltfläche <strong>Reset</strong> setzt alle Filter zurück (außer dem automatisch erkannten Spieler).</p>
<div class="admonition note">
<p>Die Filter werden in der Konfiguration von blunderDB (<code>config.yaml</code>) gespeichert und beim nächsten Start wiederhergestellt.</p>
</div>
<h4>Umschalter PR / MWC</h4>
<p>Die Schaltfläche <strong>PR / MWC</strong> am oberen Rand des Panels schaltet die in allen Tabs angezeigte Kennzahl um.</p>
<p><strong>PR (Performance Rating)</strong></p>
<blockquote>
<p>Der durchschnittliche Equity-Fehler pro gezählter Entscheidung, multipliziert mit 500 wie bei eXtreme Gammon und GNUbg: Ein PR von 5,0 entspricht 0,010 verlorener Equity pro Entscheidung, also 10 Millipunkten (mpt). Die genaue Zählregel — welche Entscheidungen in den Nenner eingehen, wie der Spielstand umgerechnet wird — ist die von Anhang: Statistikmodell — Abgleich XG / gnuBG / blunderDB.</p>
<p>Die Niveaubänder, die das Panel hinter der Fortschrittskurve zeichnet, sind ein <strong>blunderDB-eigener, indikativer Anhaltspunkt</strong>: Keine Publikation ist für diese Schwellenwerte maßgeblich. Die Obergrenze jedes Bandes ist ausgeschlossen: Ein PR von 4 ist <em>Fortgeschritten</em>, nicht <em>Experte</em>.</p>
<table>
<thead>
<tr>
<th>Niveau</th>
<th>PR</th>
</tr>
</thead>
<tbody>
<tr>
<td>Weltklasse</td>
<td>&lt; 2</td>
</tr>
<tr>
<td>Experte</td>
<td>2 – 4</td>
</tr>
<tr>
<td>Fortgeschritten</td>
<td>4 – 6</td>
</tr>
<tr>
<td>Mittelstufe</td>
<td>6 – 9</td>
</tr>
<tr>
<td>Gelegenheitsspieler</td>
<td>9 – 12</td>
</tr>
<tr>
<td>Anfänger</td>
<td>≥ 12</td>
</tr>
</tbody>
</table>
</blockquote>
<p><strong>MWC cost (Match Winning Chance cost)</strong></p>
<blockquote>
<p>Kumulierte Matchgewinnwahrscheinlichkeit, die aufgrund der Fehler über den gesamten gefilterten Datensatz verloren geht. Berechnet anhand der in blunderDB eingebetteten MET Kazaross-XG2.</p>
<div class="admonition caution">
<p>Der MWC cost <strong>ist nicht anwendbar</strong> auf <em>Money-Game</em>-Stellungen (ohne Matcheinsatz). Diese Stellungen werden von der MWC-Berechnung ausgeschlossen. Die MWC-Werte hängen von der verwendeten MET ab; sie sind nicht direkt zwischen Programmen vergleichbar, die unterschiedliche METs verwenden.</p>
</div>
</blockquote>
<p>Das Umschalten PR ↔ MWC erfolgt sofort: Es wird keine Neuberechnung im Backend durchgeführt.</p>
<h4>Der HTML-Bericht</h4>
<p>Die Schaltfläche <strong>HTML-Bericht</strong> in der Kopfzeile des Panels erzeugt ein <strong>eigenständiges</strong> Dokument: eine einzige Datei, ohne externes Bild, ohne entferntes Stylesheet, ohne Skript. Die Diagramme sind eingebettetes SVG, gezeichnet vom selben Renderer wie das Brett auf dem Bildschirm, mit Ihrer Palette. Es öffnet sich in jedem Browser, reist per E-Mail und <strong>wird vom Browser selbst als PDF gedruckt</strong> — was es erspart, einen PDF-Generator mitzuliefern für etwas, das ohnehin jeder hat.</p>
<p>Er enthält die Kennzahlen des aktuellen Bereichs (Stellungen, Matches, gezählte Entscheidungen, PR gesamt, Steine und Würfel), dann die <strong>zehn teuersten Entscheidungen</strong>, jede mit ihrem Diagramm, ihren Kosten, dem Match, aus dem sie stammt, und dem besten Zug, sofern eine Analyse ihn nennt.</p>
<p>Der Bericht trägt den <strong>aktuellen Filter</strong> des Statistik-Panels. Ein Bericht, der seinen Bereich nicht nennt, ist ein Bericht, dessen Zahlen nichts bedeuten: setzen Sie den Filter — ein Turnier, ein Zeitraum, ein Spieler — bevor Sie ihn erzeugen.</p>
<h4>Tab Dashboard</h4>
<p>Der Tab <strong>Dashboard</strong> gibt eine zusammenfassende Übersicht über die Schlüsselkennzahlen.</p>
<h5>Niveau-Karten</h5>
<p>Drei Karten zeigen den PR (oder MWC) für:</p>
<ul>
<li><strong>PR gesamt</strong> — alle Entscheidungen (Steine + Doppler);</li>
<li><strong>PR Steine</strong> — nur gespielte Steinzüge;</li>
<li><strong>PR Doppler</strong> — nur Doppler-Entscheidungen.</li>
</ul>
<p>Ein Klick auf eine Karte lädt die Stellungen der entsprechenden Teilmenge in das Analyse-Panel (Drill-down).</p>
<div class="admonition note">
<p>Die Gesamtzahl der Entscheidungen wird beim Überfahren am unteren Rand jeder Karte angezeigt.</p>
</div>
<h5>Gleitender PR über die letzten N Entscheidungen</h5>
<p>Eine Zeile mit PR- (oder MWC-)Werten, die über die letzten <em>N</em> Entscheidungen berechnet werden (N = 5, 10, 50, 100, 250, 500, 1000), ermöglicht es, den jüngsten Trend zu messen. Die ausgegrauten Werte entsprechen einem N, das größer als die Anzahl der verfügbaren Entscheidungen ist.</p>
<p>Ein Klick auf einen Wert lädt die entsprechenden letzten <em>N</em> Stellungen.</p>
<h5>Top-Blunder</h5>
<p>Die Liste der 10 schlimmsten Fehler (oder MWC cost), sortiert nach absteigender Größe. Ein Klick auf eine Zeile lädt die betreffende Stellung in das Analyse-Panel.</p>
<h4>Tab Progression</h4>
<p>Der Tab <strong>Progression</strong> zeigt die Entwicklung des Niveaus im Zeitverlauf.</p>
<p>Am Kopf des Reiters ein <strong>Ziel</strong>: „PR &lt; 5 binnen zwölf Wochen“. Ein Ziel, eine Frist und ein Trend, der sagt, wohin es geht — mehr nicht. Ein Ziel, das anfinge zu benoten, zu beglückwünschen oder zu erinnern, wäre eine andere Funktion, nicht diese.</p>
<p>Die Schaltfläche <strong>Vorschlagen</strong> schlägt ein Ziel aus dem aktuellen Niveau vor: die Untergrenze des Bandes, in dem Sie sind, also den Eintritt ins nächste. „Ein bisschen besser“ vorzuschlagen wäre an nichts verankert; ein Band vorzuschlagen sagt etwas — von fortgeschritten zu Experte zu wechseln sieht man und erzählt man.</p>
<p>Der <strong>Trend</strong> ist eine Ausgleichsgerade über das PR Ihrer Matches, auf die Frist hochgerechnet. Er weigert sich, unter drei Matches etwas zu sagen: eine Gerade zwischen zwei Punkten wäre eine Behauptung, die man nicht halten kann. Und der Satz sagt es jedes Mal — <em>ein Trend ist keine Vorhersage</em>.</p>
<p>Das Ziel wird in den <strong>Metadaten der Datenbank</strong> gespeichert, nicht in der Konfiguration: es betrifft diese Bibliothek und folgt daher der Datei statt der Maschine. Keine Schemaänderung: <code>metadata</code> ist bereits eine Schlüssel/Wert-Tabelle, lesbar von <code>blunderdb info</code> wie vom Daemon.</p>
<h5>Liniendiagramm pro Turnier</h5>
<p>Ein Liniendiagramm zeigt den PR (oder MWC) für jedes Turnier (X-Achse: Reihenfolge der Turniere, Y-Achse: Wert der Kennzahl). Farbige Bänder verdeutlichen die Niveauschwellen.</p>
<p>Ein Klick auf einen Punkt im Diagramm öffnet ein Kontextmenü mit zwei Optionen:</p>
<ul>
<li><strong>Turnier öffnen</strong> — öffnet das Turnier im Turnier-Panel.</li>
<li><strong>Positionen öffnen</strong> — lädt alle Stellungen des Turniers in das Analyse-Panel.</li>
</ul>
<h5>Streudiagramm pro Match</h5>
<p>Ein Streudiagramm stellt jedes Match dar (X-Achse: Datum, Y-Achse: PR oder MWC). Die Größe des Punkts ist proportional zur Anzahl der Entscheidungen im Match.</p>
<p>Ein Klick auf einen Punkt öffnet ein Kontextmenü:</p>
<ul>
<li><strong>Match öffnen</strong> — öffnet das Match im Matches-Panel.</li>
<li><strong>Positionen öffnen</strong> — lädt alle Stellungen des Matches in das Analyse-Panel.</li>
</ul>
<h4>Tab Fehler</h4>
<p>Der Tab <strong>Fehler</strong> schlüsselt die Fehlerquellen auf.</p>
<h5>Aufteilung nach Doppler-Aktion</h5>
<p>Ein Balkendiagramm zeigt den PR (oder MWC) für jeden Typ von Doppler-Entscheidung an: <em>NoDouble</em>, <em>DoubleTake</em>, <em>DoublePass</em>, <em>TooGood</em>. Jeder Balken gibt außerdem die Anzahl der Entscheidungen und die Blunder-Rate in einem Tooltip an.</p>
<p>Ein Klick auf einen Balken lädt die zu dieser Doppler-Aktion gehörenden Stellungen, <strong>nur die mit einem Fehler</strong> (Drill-down).</p>
<h5>Richtung der Doppler-Fehler</h5>
<p>Die Aufteilung oben sagt, <em>wie viel</em> Doppler-Entscheidungen kosten; diese Tabelle sagt, in <em>welche Richtung</em> sie falsch liegen.</p>
<p>Eine Doppler-Stellung trägt zwei Entscheidungen, die von zwei verschiedenen Spielern getroffen werden und hier in zwei Zeilen dargestellt sind:</p>
<ul>
<li><strong>Anbieten</strong> — der Spieler, der den Doppler hält, doppelt oder nicht. Seine Fehler sind die <strong>verpassten Doppel</strong> (es hätte gedoppelt werden müssen) und die <strong>verfrühten Doppel</strong> (es hätte nicht sein dürfen).</li>
<li><strong>Antworten</strong> — der Spieler, dem der Doppler angeboten wird, nimmt an oder gibt auf. Seine Fehler sind die <strong>falschen Pässe</strong> (eine korrekte Annahme wurde weggegeben) und die <strong>falschen Annahmen</strong> (ein korrekter Pass wurde angenommen).</li>
</ul>
<p>Die beiden Zeilen bleiben bewusst getrennt: Ein Spieler kann ohne Weiteres spät doppeln <em>und</em> großzügig annehmen, und eine einzige Kennzahl würde das „ausgeglichen“ nennen und dabei beide Hälften der Information verlieren.</p>
<p>Jede Zelle zeigt die Zahl der Entscheidungen; der Tooltip nennt die insgesamt verlorene Equity. Ein Klick auf eine Zelle lädt die zugehörigen Stellungen. Eine Zelle mit null ist nicht anklickbar.</p>
<div class="admonition note">
<p>Diese Tabelle zählt Entscheidungen, sie fällt kein Urteil. Ab welchem Abstand eine Tendenz einen Namen verdient, hängt von der Stichprobengröße und von einem Bezugspunkt ab — beides sind keine Daten der Engine.</p>
</div>
<h5>Aufteilung Checker / Cube</h5>
<p>Ein Vergleichsdiagramm stellt den PR der Steinzüge und der Doppler-Entscheidungen nebeneinander dar. Ein Klick auf einen Balken lädt die Stellungen der Teilmenge mit Fehler.</p>
<h5>Histogramm der Fehlergrößen</h5>
<p>Ein Histogramm verteilt die Fehler nach ihrer Größe in Millipunkten (mpt, Klassen: 0–5, 5–10, 10–25, 25–50, 50–100, ≥ 100). Ein Klick auf einen Balken lädt die Stellungen der Klasse.</p>
<h4>Registerkarte Aufschlüsselungen</h4>
<p>Der Reiter <strong>Aufschlüsselungen</strong> teilt dieselben Entscheidungen, die die Gesamtzahlen zählen, entlang vier Achsen. Keine davon definiert neu, was als Entscheidung zählt: Das wäre ein zweiter PR unter demselben Namen.</p>
<ul>
<li><strong>Nach Spielphase</strong> — Eröffnung, Mittelspiel, Wettlauf, Auswürfeln. Das beantwortet „mein PR im Wettlauf gegenüber meinem PR im Kontakt“. Die Kennzeichnung wird aus dem Brett berechnet (siehe Such-Panel); eine Datenbank, deren Phasen nie berechnet wurden, ordnet alles unter <em>Nicht klassifiziert</em> ein, und <code>blunderdb repair</code> füllt sie.</li>
<li><strong>Nach Spielplan</strong> — Rennen, Blitz, Holding, Backgame, Prime gegen Prime… Das ist die Aufschlüsselung, für die der Klassifikator existiert: „Wo verliere ich am meisten?“, Plan für Plan. Dasselbe abgeleitete Etikett wie die Phase, dieselben Vorbehalte, und <code>blunderdb repair</code> füllt es genauso.</li>
<li><strong>Nach Kennzeichnung</strong> — die <code>#Wort</code> in den Kommentaren. Eine Stellung kann mehrere tragen: <strong>diese Zeilen summieren sich nicht zur Gesamtzahl</strong>, und das Panel sagt es unter der Tabelle. Eine Kennzeichnung beschriftet, sie partitioniert nicht.</li>
<li><strong>Nach Stand</strong> — die Restpunkte beider Seiten, gelesen von der Seite des Spielers am Zug, also von der Seite dessen, der entscheidet. Die Zeile <em>Money</em> ist das Geldspiel. Eine Zelle mit weniger als zehn Entscheidungen wird <strong>ausgegraut, mit sichtbarer Anzahl</strong>, statt versteckt: zu wenige zum Lesen, aber die Auslassung bleibt überprüfbar.</li>
</ul>
<div class="admonition note">
<p>Die Crawford-Partie wird nicht unterschieden: blunderDB hält dieses Merkmal an einer Stellung nicht fest. Die praktische Wirkung ist gering — eine Crawford-Partie hat überhaupt keine Dopplerentscheidung — aber die Auslassung ist real und wird besser aufgeschrieben als dem Raten überlassen.</p>
</div>
<h4>Lernen und reales Spiel</h4>
<p>Der Befehl <code>blunderdb list --type study --days 30</code> stellt drei Zahlen nebeneinander, Spielplan für Spielplan: wie viele <strong>verschiedene Stellungen</strong> im Zeitraum wiederholt wurden, wie der PR <strong>davor</strong> war, wie der PR <strong>seither</strong> ist.</p>
<p>Drei Zahlen, und keine vierte. Es gibt <strong>keine Gewinnspalte und keinen Pfeil</strong>, denn nichts hier kontrolliert irgendetwas: Der Spieler kann stärkere Gegner getroffen, das Format gewechselt oder schlicht mehr Rennen gespielt haben. Die Zusammenschau ist Sache des Lesers; eine Spalte, die einen Effekt verkündet, würde eine Kausalität behaupten, die diese Daten nicht tragen. Die Zahlen selbst sind exakt.</p>
<p>Wiederholungen werden als <strong>verschiedene Stellungen</strong> gezählt: Eine viermal im Monat wiederholte Karte ist eine gelernte Stellung, und die Wiederholungen mitzuzählen ließe einen Monat Pauken wie einen Monat Abdeckung aussehen. Die Entscheidungen des PR dagegen zählen alle — jede wurde einmal getroffen. Ein PR auf weniger als zehn Entscheidungen zeigt <code>—</code>, mit sichtbarer Stichprobe daneben.</p>
<h4>Reiter Spieler</h4>
<p>Die vier vorigen Registerkarten beschreiben <strong>einen</strong> Spieler; die Registerkarte <strong>Spieler</strong> vergleicht sie alle. Sie zeigt eine Zeile je Spieler der Datenbank, was dem Bedarf eines Organisators entspricht, der ein ganzes Turnier verfolgt statt eines Spielers.</p>
<p>Spalten, der Reihe nach:</p>
<table>
<thead>
<tr>
<th>Spalte</th>
<th>Bedeutung</th>
</tr>
</thead>
<tbody>
<tr>
<td>Spieler</td>
<td>Der Name <strong>so, wie er in den Matches steht</strong>. Ein unter zwei Schreibweisen erfasster Spieler erscheint daher in zwei Zeilen; nutzen Sie das Zusammenführen von Spielern, um sie zu vereinen.</td>
</tr>
<tr>
<td>Matches</td>
<td>Zahl der im gewählten Zeitraum gespielten Matches.</td>
</tr>
<tr>
<td>S–N</td>
<td>Siege und Niederlagen. Ein unvollendetes Match (abgeschnittenes Protokoll, Aufgabe) zählt weder als das eine noch als das andere: S + N kann daher kleiner sein als die Zahl der Matches.</td>
</tr>
<tr>
<td>Entscheidungen</td>
<td>Zahl der gezählten Entscheidungen — der Nenner des PR. Das ist die Spalte, die sagt, was die benachbarten Kennzahlen wert sind: Ein über zwölf Entscheidungen berechneter PR bedeutet nichts.</td>
</tr>
<tr>
<td>PR</td>
<td>Gesamt-Performance-Rate.</td>
</tr>
<tr>
<td>Steine-PR, Doppler-PR</td>
<td>Der PR nach Entscheidungstyp aufgeteilt.</td>
</tr>
<tr>
<td>Snowie</td>
<td>Snowie Error Rate (siehe Anhang: Statistikmodell — Abgleich XG / gnuBG / blunderDB).</td>
</tr>
<tr>
<td>Blunders</td>
<td>Zahl der schweren Fehler (mindestens 0,100 EMG).</td>
</tr>
<tr>
<td>Glück</td>
<td>Durchschnittliches Glück pro Wurf, in Millipunkten (mpt), mit Vorzeichen: positiv, wenn die Würfel günstig waren.</td>
</tr>
</tbody>
</table>
<p>Verwendung:</p>
<ul>
<li><strong>Sortieren</strong> — auf eine Spaltenüberschrift klicken. Die Tabelle öffnet sich nach aufsteigendem PR sortiert, bester Spieler zuerst. Spieler, bei denen nichts gemessen wurde, bleiben unabhängig von der Sortierrichtung unten: eine Null mangels Daten ist keine perfekte Leistung.</li>
<li><strong>Details eines Spielers öffnen</strong> — auf eine Zeile klicken. Der Spieler wird in der Filterleiste ausgewählt, und die Anzeige wechselt zum Reiter Dashboard.</li>
<li><strong>Zeitraum einschränken</strong> — die Filter für Datum, Turnier und Matchlänge gelten wie gewohnt, wodurch sich die Tabelle auf die Tage eines Turniers begrenzen lässt.</li>
</ul>
<div class="admonition note">
<p>In diesem Reiter sind die Liste <strong>Spieler</strong> und die Wahl des <strong>Entscheidungstyps</strong> deaktiviert: Die Tabelle zeigt alle Spieler und teilt Steine- und Doppler-Entscheidungen bereits in getrennte Spalten auf.</p>
</div>
<div class="admonition important">
<p>Ein Gedankenstrich („—“) steht für einen <strong>nie gemessenen</strong> Wert, nicht zu verwechseln mit null. Das gilt insbesondere für die Spalte Glück bei jedem Match, das vor Schemaversion 2.15.0 importiert wurde: Das Glück wurde damals nicht gespeichert, und nichts erlaubt es, es nachträglich zu rekonstruieren — die Quelldateien müssen neu importiert werden. Formate, die es nicht transportieren (BGF, Jellyfish <code>.mat</code>), werden es nie liefern.</p>
</div>
<h4>Aggregationsregel</h4>
<div class="admonition important">
<p>Der PR eines Turniers (oder einer beliebigen Teilmenge) wird nach der <strong>Summe/Summe</strong>-Regel berechnet — niemals als Durchschnitt der einzelnen Match-PRs.</p>
<p>Formel:</p>
<pre class="math">PR_&#123;Turnier&#125; = 500 \\times \\frac&#123;\\sum_&#123;i&#125; \\text&#123;Fehler&#125;_i&#125;&#123;\\text&#123;Gesamtzahl der Entscheidungen&#125;&#125;</pre>
<p><strong>Beispiel:</strong> ein Spieler bestreitet zwei Matches in einem Turnier —</p>
<ul>
<li>Match A: 10 Entscheidungen, 0,100 verlorene Equity → PR = 5,0</li>
<li>Match B: 90 Entscheidungen, 0,540 verlorene Equity → PR = 3,0</li>
</ul>
<p>Naiver Durchschnitt der PRs: (5,0 + 3,0) / 2 = <strong>4,0</strong> <em>(falsch)</em></p>
<p>Summe/Summe-Regel: 500 × 0,640 / (10 + 90) = <strong>3,2</strong> <em>(korrekt)</em></p>
<p>Die Summe/Summe-Regel ist die einzige, die mit unterschiedlichen Matchlängen korrekt umgeht (ein Match über 21 Punkte wiegt mehr als ein Match über 1 Punkt).</p>
</div>
<h4>MWC: Einschränkungen</h4>
<ul>
<li>Der MWC cost wird aus der <strong>MET Kazaross-XG2</strong> berechnet, der De-facto-Referenztabelle im kompetitiven Backgammon. Die Ergebnisse sind nicht direkt mit Software vergleichbar, die andere METs verwendet. Es ist dieselbe Tabelle, gelesen über denselben Einstiegspunkt, die der eingebaute Evaluator für seine Dopplerentscheidungen beim Spielstand verwendet: Statistiken und Engine können hier nicht auseinanderlaufen. Sie liefert ihre eigenen Werte bis zu 25 zu spielenden Punkten auf jeder Seite; darüber hinaus wird sie durch eine wie bei GNUbg berechnete Zadeh-Tabelle bis 64 fortgesetzt.</li>
<li><em>Money-Game</em>-Stellungen (ohne Matchstand) werden von der MWC-Berechnung <strong>ausgeschlossen</strong>. Wenn Ihre Datenbank viele Money-Game-Stellungen enthält, kann der MWC cost unterschätzt oder nicht verfügbar sein.</li>
<li>Der MWC cost ist kumulativ über den gesamten gefilterten Datensatz — keine Kennzahl pro Entscheidung. Er misst die Gesamtauswirkung Ihrer Fehler auf Ihre Gewinnchancen.</li>
</ul>
<h3>Eval-Panel</h3>
<p>Das Panel <strong>Eval</strong> (<em>STRG-E</em>) bewertet live jede Stellung, die auf dem Brett steht, ganz gleich welche; bei einer Ausspielstellung spezialisiert es sich und berechnet zusätzlich den EPC (Effective Pip Count). Es wird aktiviert, indem man <em>STRG-E</em> drückt, auf den Reiter Eval im unteren Panel klickt, oder den Befehl <code>epc</code> ausführt. Dieser Befehl behält seinen ursprünglichen Namen: Das Panel hieß zunächst <em>EPC</em>, dann <em>Bearoff</em>, bevor es zu <em>Eval</em> wurde — hier ist also zu suchen, was eine frühere Version das Bearoff-Panel nannte, wobei der Name nur noch den Konfigurationsreiter der Ausspieltabellen bezeichnet.</p>
<p>Das Panel zeigt stets die <strong>einzige Entscheidung</strong>, die die auf dem Brett gelegte Stellung verlangt — nie zwei zugleich — und die dazugehörigen Fakten. Jede Größe wird in der Achse gelesen, die zu ihr passt, statt in einer einzigen aufgezwungenen Achse: Die Gewinn-, Gammon- und Backgammon-Wahrscheinlichkeit und die Cubeless-Equity jedes Spielers, <em>vor dem Wurf</em> berechnet, werden <strong>je Spieler</strong> gelesen (unten, oben, dann Δ), links von der Doppler-Entscheidung, solange keine Würfel gelegt sind. Fakten und Entscheidung bleiben nebeneinander: Die Doppler-Entscheidung rutscht nie unter die Zahlen, die sie begründen, unabhängig von der Sprache der Oberfläche und der Stellung auf dem Brett. Sobald Würfel gelegt sind, wechseln dieselben Werte <em>vor dem Wurf</em> die Achse: Sie werden <strong>für die Seite am Zug</strong> gelesen, am Kopf der Liste der Kandidatenzüge, als kursive Zeile <em>vor dem Wurf</em> — kein weiterer Kandidatenzug, sondern ein Bezugspunkt, an dem jeder Zug gemessen wird. Der Abstand zwischen dieser Zeile und einem Zug enthält das Glück des Wurfs, nie das Verdienst des Zugs, und sie trägt deshalb keine Fehlerspalte. Bei einer reinen Bearoff-Stellung trägt eine zweite Tabelle, stets <strong>je Spieler</strong> und stets vorhanden, mit oder ohne gelegte Würfel, den EPC, den Pip-Count, das Wastage, die durchschnittliche Wurfzahl und die Standardabweichung; diese fünf Spalten wandern nie. Die beiden Tabellen sind gestapelt und teilen dasselbe Spaltenraster: gleiche Ränder, gleiche Spaltenmarken, eine einzige Spalte mit Farbpunkten — sie lesen sich wie ein einziges Objekt mit zwei Etagen. Das Regime-Badge, die Nennung der Engine (die Tiefe der letzten Bewertung steht ebenfalls dort) und das Kontrollkästchen <em>Challenge</em> bilden ein eigenes Band, rechtsbündig über den Tabellen.</p>
<p>Nur die Liste der Kandidatenzüge scrollt — auch die Zeile <em>vor dem Wurf</em> bleibt über ihr angeheftet; der Rest des Panels (Fakten, Badge, Doppler-Entscheidung) bleibt stets sichtbar, ohne besondere Einstellung der Panelgröße.</p>
<p>Die Faktentabelle und die Entscheidung werden von gammonNet berechnet, eingebettet, ohne XG und ohne gnubg. Die Berechnung folgt der Stellung, ohne je die Oberfläche einzufrieren: Eine 0-Ply-Tiefe erscheint sofort bei jeder Eingabe; dann, nach einer halben Sekunde Ruhe, ersetzt sie eine tiefere Bewertung im Hintergrund (standardmäßig 2 Ply, einstellbar im Reiter <em>gammonNet</em> der Einstellungen) — jede neue Eingabe bricht diese Hintergrundberechnung ab. Die im Badge-Band angezeigte Tiefe, oder bei einer Rennstellung innerhalb des Regime-Badges, ist stets diejenige, die den gezeigten Wert tatsächlich erzeugt hat, nie die angeforderte; sie wird nicht in jeder Zeile wiederholt, da eine Live-Bewertung für alle Züge dieselbe Tiefe teilt. Die Equity der Kandidatenzüge und der Doppler-Entscheidung folgt dem Spielstand der Stellung: Im Money-Game wird sie in Punkten ausgedrückt, bei einem Matchstand in <strong>normalisierter Equity</strong> — derselben Skala wie XG und GNU Backgammon, auf der der Gewinn des aktuellen Dopplerwerts +1 und sein Verlust −1 zählt — nie in derselben Tabelle gemischt. Die Spaltenüberschrift nennt es ausdrücklich, statt die Skala erraten zu lassen: „Equity (money)“ im Money Game, „Equity (match)“ bei einem Matchstand. Sie berücksichtigt den <strong>lebenden Dopplerwürfel</strong>: Die Suche bewertet jede Endstellung mit dem Dopplermodell (Janowski, gemessene Effizienz) im Dopplerzustand der Stellung, wie es XG und GNU Backgammon in der <em>cubeful</em>-Bewertung tun. Das macht beim Spielstand die Effekte Gammon-Go und Gammon-Save sichtbar — bei 4-away/2-away zieht der zurückliegende Spieler bei einer Eröffnung 6-4 8/2 6/2, weil sein frühes Doppeln dem Gammon den Wert des Matches gibt, was eine Bewertung ohne Doppler nicht sehen kann. Die Zeile <em>vor dem Wurf</em> bleibt dagegen eine <strong>Cubeless-Equity</strong>: Das ist eine Tatsache der Stellung, keine Entscheidung. Dieses Panel verändert nie die Datenbank: Es ist eine Berechnung, keine gespeicherte Analyse. Ein Klick auf einen Kandidatenzug zeigt ihn als Pfeile auf dem Brett, genau wie im Analyse-Panel. Die dezente Schaltfläche <strong>?</strong> im Badge-Band führt zum Repository der Engine <code>gammonNet &lt;https://github.com/kevung/gammonNet&gt;</code>_; die vollständige Nennung (Strehl-Netz, gammonNet-Konfiguration) steht in den Danksagungen der Hilfe.</p>
<p>Der Benutzer bearbeitet die Stellung der Steine auf dem gesamten Brett, genau wie im Bearbeitungsmodus: Linksklick setzt einen Stein des unteren Spielers, Rechtsklick einen Stein des oberen Spielers. Die zweite Tabelle, die des Rennens, erscheint nur, wenn die entstandene Stellung ein reines Bearoff ist (alle Steine beider Spieler in ihrem Heimfeld); bei jeder anderen Stellung antwortet nur die Tabelle der vier gemeinsamen Spalten (Gewinn, Gammon, Backgammon, Cubeless), und die Entscheidung betrifft die Steine oder einen generischen Doppler, je nachdem, ob Würfel gelegt sind.</p>
<p>In jeder Faktentabelle eine Zeile pro Spieler — gekennzeichnet durch seinen Farbpunkt, der schwarze Spieler steht immer unten. Die erste trägt, solange keine Würfel gelegt sind, Gewinn, Gammon, Backgammon (Wahrscheinlichkeiten, ohne das %-Zeichen) und die Cubeless-Equity des Spielers; die zweite, bei einer Bearoff-Stellung und mit oder ohne gelegte Würfel, den EPC, den Pip-Count, das Wastage (Differenz zwischen EPC und Pip-Count), die durchschnittliche Wurfzahl und die Standardabweichung. Wenn beide Spieler vergleichbare Werte haben, gibt eine Zeile <strong>Δ</strong> die <em>vorzeichenbehafteten</em> Differenzen an (unten − oben: negativ, wenn der schwarze Spieler vorne liegt). Außerhalb einer Rennstellung lässt das Legen von Würfeln daher die Faktentabellen selbst verschwinden: Die vier Spalten, die sie trugen, haben soeben die Achse gewechselt, zur Seite am Zug, an den Kopf der Zugliste.</p>
<p>Die Doppler-Entscheidung hat immer dieselbe Form, woher die Zahlen auch stammen — exakte Tabelle, bewertetes Regime oder gewöhnliche gammonNet-Bewertung: <strong>eine Zeile pro Option</strong>, in der Reihenfolge <em>kein Doppeln</em>, <em>Doppeln/Take</em>, <em>Doppeln/Pass</em>, mit ihrer Equity im Bezugssystem der Stellung und ihrem Abstand zur besten Option. Die Reihenfolge ändert sich nie, anders als bei der Zugliste: Die drei Optionen tragen einen Namen, man liest also den Namen, nicht den Rang. Die beste erkennt man an ihrer Hervorhebung und an ihrer leer gelassenen Abstandszelle. Wurde der Doppler bereits gedreht, lauten die Optionen <em>kein Redoppeln</em>, <em>Redoppeln/Take</em>, <em>Redoppeln/Pass</em>.</p>
<p>Eine letzte Zeile gibt das <strong>Urteil</strong>. Es nimmt vier Werte an: <em>kein Doppeln</em>, <em>Doppeln, nehmen</em>, <em>Doppeln, passen</em> und <em>zu gut zum Doppeln</em>, Letzteres, wenn das Weiterspielen der Stellung mehr einbringt als das Kassieren des Punktes: Doppeln wäre dann ein Fehler aus dem umgekehrten Grund wie beim einfachen <em>kein Doppeln</em>. Es ist auch die einzige Stelle, an der das Panel sagt, dass es <strong>kein</strong> Urteil gibt, statt eine laufende Berechnung vorzutäuschen:</p>
<ul>
<li><em>keine Entscheidung</em> — das Regime hat keinen Anspruch darauf; das Würfel-Urteil wird nie geschätzt (siehe das Badge <em>geschätzt</em>);</li>
<li><em>bei diesem Stand nicht auswertbar</em> — die Engine lehnt die Stellung ab, typischerweise ein Spielstand außerhalb des Horizonts der Match-Equity-Tabelle, also eine Seite mit mehr als 64 zu spielenden Punkten;</li>
<li><em>gegnerischer Doppler</em> und <em>Doppler tot (Crawford)</em> — der Doppler kann nicht gedreht werden. Die Equities bleiben zur Orientierung angezeigt, aber keine Option trägt einen Abstand: Ein Fehler ist das, was eine Wahl kostet, und es gibt keine Wahl.</li>
</ul>
<p>Im Money Game erscheinen die auf der Position aktiven Regeln <strong>Jacoby</strong> und <strong>Beaver</strong> unter der Doppelwürfeltabelle, als kleine Badges neben dem Urteil, das sie verändern: Das Urteil „kein Doppeln“ einer Position unter der Jacoby-Regel ist nicht dieselbe Berechnung wie ohne sie, und nichts anderes auf dem Bildschirm sagte das aus.</p>
<p>Ein drittes Abzeichen, <strong>Max. Verdopplung</strong>, erscheint, wenn die Quellkennung den Verdopplungswürfel deckelt — bei Matchstand ebenso wie im Money Game. Dieses beschreibt nicht die darüber gezeigte Berechnung: Der eingebaute Evaluator modelliert keine Obergrenze, das Urteil gilt also für einen freien Verdopplungswürfel. Genau deshalb steht das Abzeichen dort: Ein gedeckelter Verdopplungswürfel ist der eine sichtbare Grund, aus dem blunderDB und eXtreme Gammon zur selben Stellung zwei verschiedene Urteile verkünden können.</p>
<p>Das Regime-Badge, die Bewertungstiefe, der Link zur Engine und das Kontrollkästchen <em>Challenge</em> bilden ein eigenes Band, rechtsbündig über den Tabellen.</p>
<p>Der <strong>Spieler am Zug</strong> und die <strong>Position des Dopplers</strong> werden direkt auf dem Brett bearbeitet, wie im Bearbeitungsmodus: Ein Klick auf das Bearoff-/Score-Rechteck eines Spielers macht ihn zum Spieler am Zug; ein Klick auf den Doppler wechselt zyklisch zentriert → Besitz unten → Besitz oben (Rechtsklick in umgekehrter Richtung). Der Wert des Dopplers bleibt fest — im Money-Game werden die Equities in Einheiten des aktuellen Dopplers ausgedrückt, nur sein Besitzer zählt. Die Analyse wird sofort neu berechnet. Im geschätzten Regime ist das Badge selbst anklickbar und öffnet direkt den Reiter <em>Bearoff</em> der Einstellungen; sein Tooltip erklärt, warum (Würfel-Urteil nicht schätzbar, <code>ADR-0009 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0009-race-win-chances-are-read-or-convolved-cube-verdicts-are-never-estimated.md&gt;</code>__) und wie sich der exakte Bereich erweitern lässt.</p>
<p>Auch der <strong>Spielstand</strong> wird direkt auf dem Brett bearbeitet, wie im Bearbeitungsmodus: Linksklick auf das Score-Rechteck eines Spielers verringert die Zahl seiner noch benötigten Punkte, Rechtsklick erhöht sie. Verlässt man den <em>Money</em>-Spielstand (-1, -1) durch Bearbeiten nur einer Seite, wird die andere Seite automatisch auf denselben Wert gesetzt, statt einen inkonsistenten Spielstand stehen zu lassen. Bei einer Bearoff-Stellung im Regime <em>exakt</em> lässt der Wechsel von einem Money-Spielstand zu einem Matchstand die Gewinnwahrscheinlichkeit unverändert (ein Wert aus der Datenbank, gültig in jedem Bezugssystem), schaltet aber die angezeigte Equity und das Würfel-Urteil auf die des Regimes <em>bewertet</em> um — die exakte Tabelle ist konstruktionsbedingt Money und kann die beim Matchstand gestellte Frage nicht beantworten. Das Badge wird dann zusammengesetzt („exakt (Gewinn) · bewertet (Würfel)“), um dies ausdrücklich zu sagen.</p>
<p>Die <strong>Würfel</strong> schließlich werden auf dieselbe Weise bearbeitet, und sie entscheiden, welche Frage gestellt wird: gelegte Würfel ergeben eine Steinentscheidung (die Liste der Kandidatenzüge), keine Würfel eine Doppler-Entscheidung. Ein Linksklick auf einen Würfel erhöht seinen Wert (6 springt auf 1), ein Rechtsklick verringert ihn (1 springt auf 6); ein Klick auf einen Würfel auf einem Brett ohne Würfel legt zwei auf einmal — ein einzelner Würfel wäre weder eine Steinentscheidung noch eine Doppler-Entscheidung. Ein Klick auf das Rechteck eines Spielers entfernt die Würfel, um eine Doppler-Frage zu stellen, und der nächste Klick auf einen Würfel legt sie wieder so hin, wie sie waren.</p>
<p><em>RÜCKTASTE</em> oder ein Doppelklick außerhalb des Bretts löscht die Stellung: leeres Brett, Money-Spielstand (-1, -1), keine gelegten Würfel — panel-eigene Werte des Eval-Panels, die sich von denen des Bearbeitungsmodus unterscheiden (7 überall, Würfel 3-1), um mit dem übereinzustimmen, was das Panel standardmäßig anzeigt.</p>
<h4>Dopplerwürfel-Matrix</h4>
<p>Eine Dopplerwürfel-Entscheidung ist keine Eigenschaft des Bretts. Dieselben Steine, derselbe Pipstand, sind bei 2-away/4-away ein Doppel und bei 4-away/2-away keines; wer die Money-Antwort gelernt hat, hat eine Zelle eines Rasters gelernt. Das Eval-Panel zeigt die Zelle, die die Stellung trägt; die <strong>Dopplerwürfel-Matrix</strong> zeigt das ganze Raster.</p>
<p>Der Befehl <code>cm</code> öffnet sie für die angezeigte Stellung. Jede Zelle gibt das Urteil bei einem Punktestand: die Zeile ist die Zahl der Punkte, die dem Spieler am Zug noch fehlen, die Spalte die des Gegners. Die vier Urteile lauten <em>KD</em> (kein Doppel), <em>DA</em> (Doppel, Annahme), <em>DP</em> (Doppel, Pass) und <em>ZG</em> (zu gut); eine von der Engine abgelehnte Zelle trägt ein Fragezeichen und nennt beim Überfahren den Grund sowie die drei Equities der Zelle. Drei Matchlängen stehen zur Wahl: 5, 7 und 9 Punkte.</p>
<p>Der eigene Punktestand der Stellung wird durch den jeder Zelle ersetzt; ihr <strong>Dopplerwürfel</strong> bleibt erhalten. Das Raster beantwortet, bei welchem Punktestand man <em>diesen</em> Würfel drehen würde, nicht was eine zentrierte Stellung täte. Es gilt durchgehend nach Crawford: im Crawford-Spiel ist der Würfel nicht im Spiel, und eine Spalte „Sie dürfen nicht doppeln“ würde nichts über die Stellung sagen.</p>
<p>Jede Zelle ist eine eigene Suche. Die Engine berücksichtigt den Punktestand — bei 2-away spielt sie nicht dieselbe Partie wie bei 7-away —, also wäre eine einzige, durch verschiedene Match-Equities gelesene Suche genau dort falsch, wo der Punktestand zählt. Das Raster erscheint zuerst mit 0-Ply und wird dann in der eingestellten Anzeigetiefe neu berechnet, sobald das Fenster ruht: dieselbe Eskalation wie im Rest des Panels, für ein 9-Punkte-Raster von etwa anderthalb Sekunden.</p>
<p>Dasselbe Raster wird außerhalb der Oberfläche mit dem Befehl cubematrix der Kommandozeile berechnet.</p>
<h4>Eine Stellung in das Eval-Panel bringen</h4>
<p>Das Panel öffnet sich standardmäßig mit einer Bearoff-Stellung, doch die Untersuchung geht meist von einer bereits vorliegenden Stellung aus. Zwei Gesten bringen sie dorthin:</p>
<ul>
<li><strong>Rechtsklick auf das Brett</strong>, in einem Analyse-Panel oder während der Navigation in einem Match, dann <em>Diese Position auswerten</em>: Das Eval-Panel öffnet sich direkt mit dieser Stellung, so wie sie angezeigt wird. Das Kontextmenü erscheint weder im Eval-Panel noch im Suchpanel, wo die rechte Maustaste bereits zum Setzen der Steine der anderen Farbe dient.</li>
<li><strong>CTRL-C, dann CTRL-V</strong>: die Stellung aus dem Analyse-Panel kopieren und dann einmal im Eval-Panel einfügen. Das Einfügen akzeptiert auch eine Kennung von anderswo — eine XGID (eXtreme Gammon, GNU Backgammon, eine andere blunderDB-Instanz) oder eine OGID (OpenGammon): Sie muss nur in der Zwischenablage liegen.</li>
<li><strong>Der Befehl</strong> <code>import XGID=…</code> (oder <code>import OGID=…</code>) für den Fall, dass die Kennung nicht in der Zwischenablage steht, sondern in einer Nachricht, in einem im Terminal gelesenen Forum oder von einem Skript erzeugt wurde. Es ist derselbe Befehl wie <code>import</code> allein: ohne Argument öffnet er eine Dateiauswahl, mit Argument liest er die Kennung. Der Weg ist danach mit dem des Einfügens identisch — dieselbe Lesung, dieselbe Deduplizierung, dasselbe Öffnen der importierten Stellung.</li>
</ul>
<p>Eine OGID trägt nur eine Stellung: keine Auswertung, keinen Kommentar. Die Stellung kommt also ohne Analyse an, genau wie eine nackte XGID, und der eingebaute Evaluator kann die Lücke danach füllen.</p>
<p>Das Brett des Eval-Panels ist ein Entwurf: Die Stellung kommt ohne ihre Datenbankkennung dort an, sodass keine hier vorgenommene Änderung den Datensatz überschreiben kann, aus dem sie stammt. Alle üblichen Bearbeitungen des Bretts bleiben dort verfügbar (Steine, Doppler, Würfel, Spielstand), und die Bewertung folgt jeder Änderung.</p>
<p>In der Gegenrichtung kopiert <em>CTRL-C</em> das Brett des Eval-Panels in die Zwischenablage, mit einer aus den gesetzten Steinen neu berechneten XGID — also direkt in eXtreme Gammon oder in eine andere blunderDB-Instanz einfügbar. Nur die Stellung reist mit: Die vom Panel angezeigte Bewertung ist kein Datensatz der Datenbank und begleitet die Kopie nicht.</p>
<p>Beim Verlassen des Eval-Panels wird die zuvor betrachtete Stellung wiederhergestellt: Der Entwurf wird nie von selbst gespeichert.</p>
<p>Wenn die Stellung ein reines Bearoff ist (alle Steine beider Spieler in ihrem Heimfeld) und keine Würfel gelegt sind, zeigt die Doppler-Entscheidung für den Spieler am Zug:</p>
<ul>
<li>im Regime <em>exakt</em>: die Money-Equities (Cubeless, ohne Doppeln, Doppeln/Take, Doppeln/Pass) und das <strong>Money-Würfel-Urteil</strong> (kein Doppeln, Doppeln/Take, Doppeln/Pass oder zu gut zum Doppeln) — außerhalb eines Matchstands, siehe oben für den Fall des Matchstands,</li>
<li>im Regime <em>bewertet</em>: dieselben Equities und dasselbe Urteil mit vier Werten, aber <strong>von gammonNet gespielt</strong> (Suche + Janowski-Dopplermodell) statt aus einer Tabelle gelesen — verfügbar <strong>auch beim Matchstand</strong>, was das geschätzte Regime nie bieten konnte;</li>
<li>im Regime <em>geschätzt</em>: Das Würfel-Urteil wird dann bewusst nicht angezeigt — nur die Gewinnwahrscheinlichkeit in der Faktentabelle, begleitet von ihrer Fehlermarge, bleibt verfügbar.</li>
</ul>
<p>Sobald auf einer Rennstellung Würfel gelegt sind, verschwindet diese Doppler-Entscheidung <em>vor dem Wurf</em> — das Brett verlangt dann eine Zugentscheidung, keine Doppler-Entscheidung —, doch die Gewinnwahrscheinlichkeit bleibt ein Fakt der Stellung, keine Entscheidung: Sie wandert in die Zeile <em>vor dem Wurf</em> am Kopf der Zugliste, neben den EPC, der seinerseits direkt links davon angezeigt bleibt.</p>
<p>Ein Badge zeigt das Regime an: <strong>exakt</strong> (Wert aus einer zweiseitigen Datenbank gelesen), <strong>bewertet · &lt;Tiefe&gt;</strong> (von gammonNet gespielt — die angezeigte Tiefe ist diejenige, die den gezeigten Wert tatsächlich erzeugt hat), <strong>geschätzt ± Marge</strong> oder, beim Matchstand im exakten Bereich, <strong>exakt (Gewinn) · bewertet (Würfel)</strong> — siehe oben. Das exakte Regime hat überall Vorrang, wo es verfügbar ist; andernfalls erscheint das bewertete Regime, sobald es fertig gerechnet hat, und ersetzt an Ort und Stelle das während des Wartens gezeigte geschätzte Regime. Siehe Methodik und Annahmen des Eval-Panels für die genaue Definition der drei Regime und ihrer Annahmen.</p>
<p><strong>Die exakte Domäne erweitern.</strong> Die beim ersten Start berechnete Tabelle deckt 6 Steine je Seite ab. Zwei Wege darüber hinaus, im Reiter <em>Bearoff</em> der Konfiguration:</p>
<ul>
<li>eine breitere zweiseitige Tabelle berechnen — bis TS-06-15, wenn die Maschine den Speicher dafür hat. Der Reiter nennt Größe, Speicher und Dauer auf dieser Maschine vor dem Start, und die Berechnung lässt sich pausieren und fortsetzen. Eine abgebrochene Berechnung hinterlässt eine <code>.part</code>-Datei, die nie als Tabelle gelesen wird;</li>
<li>eine beliebige Two-Sided-<code>.bd</code>-Datei von gnubg angeben. Die Datenbank mit dem größten Bereich hat automatisch Vorrang.</li>
</ul>
<p><strong>Das Brett des Panels ist ein Entwurf, und es wird gemerkt.</strong> Wer das Eval-Panel verlässt und zurückkehrt, findet die Stellung wieder, auf der er es verlassen hat, nicht das voreingestellte Ausspielbrett: dieses wird nur beim ersten Öffnen in einer Sitzung gezeigt. Eine aus der Datenbank ins Panel geschickte Stellung sticht diese Erinnerung, und <em>RÜCKTASTE</em> stellt jederzeit das Standardbrett wieder her. Dabei wird nichts in die Datenbank geschrieben — der Entwurf hat keine Stellungsidentität, und seine Bewertung wird bei der Ankunft neu berechnet statt mitgeführt.</p>
<p><strong>Challenge-Modus.</strong> Das Kontrollkästchen <em>Challenge</em> im Badge-Band aktiviert einen Trainingsmodus: Bei jeder Änderung der Stellung werden die Werte dreier Zonen verdeckt (ersetzt durch „···“); ein Klick auf eine Zone deckt nur diese Zone auf. Ohne Würfel sind das die Zeile des unteren Spielers, die Zeile des oberen Spielers und die Doppler-Entscheidung — die Δ-Zeile erscheint erst, wenn beide Spielerzeilen aufgedeckt sind. Der Entscheidungsblock behält dabei seine drei Zeilen: Es sind seine Werte, sein Urteil und die Hervorhebung der besten Option, die verschwinden, da sich die Übung sonst durch die Suche nach der fetten Zeile lösen ließe. Sind auf einer Rennstellung Würfel gelegt, wird die EPC-Zeile jedes Spielers wie zuvor verdeckt, doch die dritte Zone umfasst dann die Zeile <em>vor dem Wurf</em> und die Zugliste <strong>zusammen</strong>: Da die Liste vom besten zum schlechtesten Zug sortiert ist, würde ihr teilweises Aufdecken bereits die Antwort verraten. Sind Würfel außerhalb einer Rennstellung gelegt, umfasst diese eine Zone allein alles, was das Panel anzeigt. So kann man üben, den EPC jeder Seite zu schätzen und sich dann zum Doppler oder zum zu spielenden Zug zu äußern, bevor man nachprüft. Die Einstellung wird gespeichert.</p>
<p>Um das Eval-Panel zu schließen, drücken Sie <em>CTRL-E</em> oder wechseln Sie zu einem anderen Tab.</p>
<h4>Methodik und Annahmen des Eval-Panels</h4>
<p>Jeder vom Panel angezeigte Wert beruht auf präzisen Annahmen, die hier erschöpfend dargelegt werden.</p>
<p><strong>Bereich.</strong> Die <em>Rennzone</em> — Gewinnwahrscheinlichkeit und Doppler-Urteil — behandelt nur reine Auswürfelstellungen: alle verbleibenden Steine beider Spieler im eigenen Heimfeld. Die Stellung wird <em>vor dem Wurf</em> bewertet; gesetzte Würfel werden ignoriert.</p>
<p>Die <strong>EPC-Blöcke</strong> gehen ihrerseits weiter: eine Seite bekommt ihren EPC, sobald ihr entferntester Stein in die geladene einseitige Tabelle passt. Mit der Standardtabelle (sechs Punkte) ist das die alte Heimfeld-Regel; mit einer Tabelle mit acht Punkten, im Reiter <em>Bearoff</em> berechnet, wird eine Seite mit einem Stein auf dem 8-Punkt behandelt wie jede andere. Nichts wird extrapoliert: ein Stein einen Punkt zu weit hat schlicht keinen EPC, genau wie ein Stein auf dem 7-Punkt zuvor keinen hatte. Ist die antwortende Tabelle nicht die mit sechs Punkten, erscheint ihr Name in der Ecke des Rennblocks („OS-08“) — ohne ihn läse man standardmäßig „sechs“ und hielte die Seite für ganz zu Hause.</p>
<p><strong>EPC-Blöcke (immer exakt).</strong> EPC, mittlere Wurfzahl und Standardabweichung stammen aus der exakten Verteilung der Wurfzahl zum Auswürfeln aller Steine, gelesen aus GNUbgs einseitiger Datenbank (6 bis 10 Punkte, 15 Steine, auf der Maschine berechnet). EPC = mittlere Würfe × 49/6 (49/6 ≈ 8,167 ist der exakte Mittelwert der Pips je Wurf, Pasch vierfach gezählt); Wastage = EPC − Pipzahl. Die einzige Idealisierung ist das <em>einseitig optimale Spiel</em>: jeder Spieler minimiert die eigenen Würfe und ignoriert den Gegner — das ist die übliche Definition des EPC.</p>
<p><strong>Gewinnwahrscheinlichkeit, exaktes Regime.</strong> Direktes Lesen aus der größten verfügbaren Two-Sided-Datenbank (TS-06-06, beim ersten Start berechnet, externe Datei, oder TS-06-11, im Reiter <em>Bearoff</em> berechnet). Diese Datenbanken sind das Ergebnis einer vollständigen Rückwärtsanalyse unter optimalem Two-Sided-Spiel beider Seiten: keine zusätzliche Annahme, Fehler beschränkt auf die Quantisierung (&lt; 0,002 %).</p>
<p><strong>Gewinnwahrscheinlichkeit, geschätztes Regime.</strong> Außerhalb des Bereichs der Datenbank: Die Wahrscheinlichkeit wird durch Faltung der beiden One-Sided-Verteilungen ermittelt (der Spieler am Zug gewinnt, wenn seine Wurfzahl kleiner oder gleich der des Gegners ist) und anschließende Anwendung einer festen polynomialen Korrektur, offline gegen die TS-06-11-Datenbank kalibriert. Drei Annahmen:</p>
<ul>
<li><strong>Unabhängigkeit</strong> der beiden Abtragprozesse — im Rennen strukturell gegeben, ohne Kontakt gibt es keinerlei Interaktion;</li>
<li><strong>optimales One-Sided-Spiel beider Seiten</strong> — das ist <em>die Approximation</em>: In Wirklichkeit weicht der zurückliegende Spieler ab, um auf Varianz zu spielen, und der Führende, um auf Sicherheit zu spielen. Der gemessene Effekt ist eine antisymmetrische Verzerrung (die Faltung übertreibt den Vorsprung des Führenden), die die Korrektur statistisch auffängt;</li>
<li>die <strong>Korrektur</strong> wurde auf dem Bereich des Orakels kalibriert und validiert (bis 11 Steine pro Spieler). Gemessener Restfehler: Standardabweichung 0,05 %, 99. Perzentil 0,17 %, beobachtetes Maximum 0,9 % (in Prozentpunkten der Gewinnwahrscheinlichkeit). <strong>Jenseits von 11 Steinen pro Spieler ist diese Schranke extrapoliert</strong> — die Tendenz ist monoton, aber kein Orakel bestätigt sie.</li>
</ul>
<p><strong>Equities und Würfel-Urteil (nur exaktes Regime).</strong> Die angezeigten Equities sind die des <strong>Money-Game ohne Jacoby</strong>, im Referenzrahmen der Bearoff-Literatur. Im Bereich ≤ 11 Steine pro Spieler sind Gammons unmöglich (jede Seite hat bereits mindestens 4 Steine abgetragen): Das ist keine Approximation. Das Urteil (kein Doppeln / Doppeln, Take / Doppeln, Pass) wird exakt aus den gespeicherten Equities rekonstruiert, nach der Regel von GNUbg, Zug für Zug gegen dessen Analyse validiert.</p>
<div class="admonition note">
<p>Die Cubeful-Equities setzen ein <strong>optimales Spiel mit dem Verdopplungswürfel beider Seiten bis zum Ende</strong> voraus: Künftige Redoubles werden vollständig bewertet (vollständige Rückwärtsanalyse). In den sehr volatilen Rennen am Partieende frisst die Kaskade der Redoubles fast den gesamten Vorteil der Seite am Zug auf — die Equities „ohne Doppeln“ und „Doppeln/Take“ können dann nahe null liegen, wo eine Engine wie XG, deren Würfelmodell diese Kaskade nicht bewertet, Werte nahe dem Dead Cube anzeigt (zum Beispiel 2 Steine auf Punkt 3 gegen 2 Steine auf Punkt 2: 62 % der Partien gewonnen, exaktes D/T +0,006 gegenüber +0,475 bei XG). Die angezeigte <strong>Entscheidung</strong> hingegen stimmt mit der der Engines überein.</p>
</div>
<p><strong>Gewinnwahrscheinlichkeit und Urteil, bewertetes Regime.</strong> Außerhalb des exakten Bereichs stammt die Gewinnwahrscheinlichkeit aus der Rohausgabe von gammonNet (Suche mit 0 oder 2 Ply je nach Geste, nie aus einer Tabelle gelesen) und das Urteil aus einem auf diese Ausgabe angewandten Janowski-„Decide“ — die Suche <em>spielt</em> die Trajektorie, statt eine Momentaufnahme davon zusammenzufassen; genau das konnte das geschätzte Regime nicht (siehe unten), und es erlaubt, als einziges der drei Regime neben dem exakten, ein Urteil <strong>beim Matchstand</strong>.</p>
<p>Dieses Regime wurde gemessen, nicht nur angenommen, gegen die eingebettete zweiseitige Tabelle (<code>TestEvalMeasure</code>, 4000 gesampelte Money-Entscheidungen, kanonische Parameter 2 Ply k=12): Übereinstimmung des Money-Urteils <strong>93,4 %</strong> (3735/4000), aufgeschlüsselt nach Abstand zum Take-Point von gammonNet — 61,1 % bei weniger als 1 % vom Take-Point (die Zone, die am empfindlichsten für einen Münzwurf ist), 88,3 % zwischen 1 und 5 %, 91,5 % zwischen 5 und 10 %, 94,0 % zwischen 10 und 20 %, 94,4 % darüber. Abweichung der Gewinnwahrscheinlichkeit: Mittelwert 0,85 %, Median 0,44 %, 95. Perzentil 3,21 %, Maximum 8,30 %. Abweichung der Cubeful-Equity: Mittelwert 0,039, Median 0,018, 95. Perzentil 0,151, Maximum 0,406. Die Form ist die erwartete: Der Großteil der Abweichung konzentriert sich genau am Take-Point, wo zwei legitim verschiedene Methoden bei einer knappen Entscheidung am stärksten auseinandergehen — kein diffuser Fehler, der überall Equity kosten würde.</p>
<p>Diese Messung bezieht sich auf <strong>Money</strong>-Entscheidungen im Rennen. Für das Urteil beim Spielstand — das nur dieses Regime zu liefern vermag — und für Kontaktstellungen liegt keine veröffentlichte Messung vor: Das Vorstehende überträgt sich nicht auf diese Fälle.</p>
<p><strong>Warum nicht tiefer als 2 Ply?</strong> Weil die Messung sagt, dass es nichts einbringt. Eine Steinentscheidung kostet auf derselben Maschine 99 ms bei 2 Ply und 8,4 s bei 3 Ply — <strong>fünfundachtzigmal mehr</strong>. Über vierzig echte, in beiden Tiefen wiederholte Entscheidungen änderte die tiefere Suche <strong>zweimal</strong> ihre Meinung, und beide Male war der Gewinn, den sie sich selbst zuschrieb, höchstens 0,0005 normalisierte Äquität: zwei Größenordnungen unter 0,020, der Schwelle, ab der eXtreme Gammon überhaupt von einem Fehler spricht. Pro Entscheidung, alle Fälle zusammen, beträgt der Gewinn 0,0000.</p>
<p>Die Einstellung wird daher nicht angeboten. Das heißt nicht, dass 3 Ply allgemein wertlos wäre, sondern nur, dass es auf <em>diesem</em> Netz, mit dem kanonischen Filter, das Warten dessen nicht bezahlt, der vor einem Panel sitzt. Die Messung ist reproduzierbar (<code>TestThreePlyMeasure</code>), und die Schlussfolgerung ist neu zu treffen, wenn sich das Netz ändert.</p>
<p><strong>Warum gibt es das geschätzte Urteil nicht?</strong> Das Folgende betrifft speziell die Methode per <em>Faltung</em> (geschätztes Regime), nicht das oben beschriebene bewertete Regime: Die Cubeful-Equity ist ein Problem der <em>Trajektorie</em> (wann doppeln?), das keine statistische Zusammenfassung der Stellung erfasst — das beste gemessene statische Modell lässt einen Restfehler (Standardabweichung 0,016 Equity, Maximum 0,20), der ausreicht, um alle knappen Entscheidungen umzukehren. Ebenso wurde die Umrechnung des Urteils auf den Matchstand über eine Match-Equity-Tabelle als unzureichend gemessen (12 % der Entscheidungen weichen von der 2-ply-Analyse von GNUbg ab, mit echten Blunders). Da ein mit Nachdruck angezeigtes falsches Urteil schlimmer ist als gar kein Urteil, durfte die Faltung nie ein Urteil anzeigen — diese Lücke füllt eine Suche, die die Trajektorie spielt, kein statistischer Abriss.</p>
<div class="admonition note">
<p>Die Bearoff-Datenbanken sind unveränderliche mathematische Tabellen. blunderDB berechnet sie selbst, identisch zu GNUbgs Werkzeug <code>makebearoff</code> — Byte für Byte — im Reiter <em>Bearoff</em> der Konfiguration oder mit <code>blunderdb bearoff generate</code>.</p>
</div>
<h3>Anki-Panel</h3>
<p>Das Panel <strong>Anki</strong> (<em>CTRL-K</em>) ermöglicht es, Stellungen durch verteiltes Wiederholen mithilfe des FSRS-Algorithmus zu studieren. Der Benutzer kann Stapel aus Sammlungen oder Suchergebnissen erstellen.</p>
<p><strong>Stapel erstellen:</strong> Klicken Sie auf <em>New Deck</em>, um einen Stapel aus einer Sammlung oder den aktuellen Suchergebnissen zu erstellen. Suchbasierte Stapel werden beim Aktivieren des Anki-Tabs automatisch synchronisiert.</p>
<p><strong>Wiederholung:</strong> Wählen Sie einen Stapel aus und klicken Sie dann auf <em>Study</em> (oder doppelklicken Sie auf einen Stapel), um mit der Wiederholung der fälligen Karten zu beginnen. Jede Karte zeigt die entsprechende Stellung auf dem Brett an. Bewerten Sie Ihre Erinnerung mit den Tasten <em>1</em> (Nochmal), <em>2</em> (Schwierig), <em>3</em> (Gut) oder <em>4</em> (Einfach). Drücken Sie <em>Esc</em>, um anzuhalten und zur Stapelliste zurückzukehren.</p>
<p><strong>Verdopplungsentscheidungen ergeben zwei Karten, verkettet.</strong> Eine Verdopplungsentscheidung ist zwei Fragen — „doppeln?“, dann „annehmen?“ — und blunderDB speichert sie seit jeher als zwei Stellungen. Ein Stapel, der nur eine Hälfte auswählt, bekommt die andere: Die Entscheidung wird vervollständigt, nicht erweitert. Und wenn beide fällig sind, kommt die zweite <strong>unmittelbar</strong> nach der ersten.</p>
<p>Jede behält ihre eigene Note und ihren eigenen Plan: Das sind nicht zwei Stufen einer Karte, das sind zwei Karten. Die Verkettung zieht keinen Termin vor — sie ordnet die bereits fälligen Karten, mehr nicht. Da beide zusammen entstehen, sind sie beim ersten Mal zusammen fällig, und genau dort nützt sie.</p>
<p><strong>Antwort anzeigen:</strong> Die Karte stellt eine Frage — welcher Zug zu spielen ist oder welche Doppler-Aktion. Überlegen Sie, und drücken Sie dann <em>LEERTASTE</em> (oder klicken Sie auf den verdeckten Bereich), um die Antwort aufzudecken: die gespeicherte Analyse der Stellung, so wie der Tab Analyse sie darstellt. Sie erscheint unter den Bewertungsschaltflächen, die an ihrem Platz und in Reichweite bleiben. Ein Klick auf einen Zug der Liste zeigt ihn auf dem Brett.</p>
<p>Nichts zwingt Sie, die Antwort aufzudecken, um zu bewerten: wenn Sie sich sicher sind, bleiben die Tasten <em>1</em> bis <em>4</em> aktiv. Die Antwort wird bei der nächsten Karte wieder verdeckt, nicht aber, wenn Sie nur den Tab wechseln — sehen Sie im Eval-Panel oder im Kommentar der Stellung nach, sie wartet bei Ihrer Rückkehr auf Sie.</p>
<p>Eine Stellung ohne gespeicherte Analyse zeigt dies direkt an, ohne verdeckten Bereich.</p>
<p><strong>Sitzung begrenzen.</strong> Standardmäßig läuft eine Wiederholungssitzung bis zum Ende der fälligen Karten. In den Einstellungen können Sie sie je Deck auf eine Kartenzahl begrenzen: Haken Sie <em>Sitzung begrenzen</em> an und geben Sie an, wie viele Karten eine Sitzung ausgeben soll. Ist die Grenze erreicht, endet die Sitzung mit einem Hinweis — die Meldung unterscheidet „Grenze erreicht, so viele Karten noch fällig“ von einer wirklich leeren Warteschlange. Wer dennoch weitermachen will, hat das freie Üben: Es zeigt andere Stellungen, ohne am Zeitplan etwas zu ändern.</p>
<p>Eine Grenze von <strong>0</strong> gibt gar keine Karte aus: Das ist ein eigenständiger Zustand, nützlich, um ein Deck während der Turniervorbereitung einzufrieren, und nicht dasselbe wie „keine Grenze“. Die Schaltfläche <em>Study</em> ist dann inaktiv.</p>
<p>Die Grenze gilt für die <strong>Sitzung</strong>, nicht für den Tag. Ein blunderDB-Deck beruht auf einer Sammlung oder einer Suche: Es ist ein endlicher Bestand, der über wenige Sitzungen eingeführt wird und dessen Tagesvolumen bereits durch seine Größe begrenzt ist. Eine Tagesgrenze würde nie greifen oder aber einen Rückstand auf einem Deck erzeugen, das in eine Sitzung passte.</p>
<p><strong>Freies Üben (Cram):</strong> Die Schaltfläche <em>Cram</em> neben <em>Study</em> startet eine freie Übungssitzung: Es werden Ihnen zufällige Stellungen aus dem Deck angezeigt, unabhängig vom FSRS-Zeitplan. Dieser Modus <strong>verändert niemals den Plan der verteilten Wiederholung</strong> — ideal zum Aufwärmen vor einem Turnier oder zum intensiven Wiederholen eines thematischen Decks, ohne dessen Reihenfolge zu stören. Eine <em>Cram</em>-Plakette ersetzt den Kartenstatus, und eine Schaltfläche <em>Weiter</em> (Tasten <em>1</em> bis <em>4</em>) blättert durch die Stellungen. <em>Esc</em> kehrt zur Liste zurück, ohne eine unterbrochene Sitzung zu speichern.</p>
<p><strong>Eine Karte beiseitelegen, ohne sie zu benoten.</strong> Während einer Wiederholung bietet ein Rechtsklick auf die Kopfzeile der Karte drei Gesten, die sie aus der Sitzung nehmen, ohne dem Planer irgendetwas zu sagen:</p>
<ul>
<li><strong>Aussetzen</strong> — die Karte behält ihren Termin und kommt, solange sie ausgesetzt ist, nie wieder dran. So legt man eine falsche oder noch nicht nützliche Karte beiseite, ohne die daran hängende Historie zu verlieren.</li>
<li><strong>Zurückstellen</strong> — die Karte verschwindet bis zum nächsten Tag. Anders als das Aussetzen sagt das nichts über ihren Wert: es ist für die, die man gerade anderswo gesehen hat oder an einem Abend nicht zweimal treffen möchte.</li>
<li><strong>Entfernen</strong> — die Karte verlässt nach Bestätigung den Stapel. Die Stellung selbst bleibt in der Datenbank: ein Stapel ist eine Lernliste über die Bibliothek, nie eine Kopie davon.</li>
</ul>
<p>Keine dieser drei Gesten erfasst eine Note: eine beiseitegelegte Karte ist keine beantwortete Karte und zählt nicht zur Sitzung.</p>
<p><strong>Wiederholungsprotokoll.</strong> In den Einstellungen eines Stapels zeigt die Schaltfläche <em>Wiederholungsprotokoll</em>, was dem Planer <strong>gesagt</strong> wurde — Datum, Stellung, Note, Zustand, gewährtes Intervall — im Gegensatz zu dem, was er vorhat. Nur hier ist eine versehentlich eingegebene Note überhaupt zu sehen. Korrigieren lässt sie sich dort nicht: der Terminplan bleibt außer Reichweite, und genau diese Regel macht das Protokoll nützlich — die Vergangenheit lässt sich nicht umschreiben, aber sie lässt sich kennen.</p>
<p><strong>Anhalten/Fortsetzen:</strong> Sie können eine Wiederholungssitzung jederzeit mit <em>Esc</em> unterbrechen. Die Schaltfläche wechselt zu <em>Resume</em> und zeigt Ihren Fortschritt an. Klicken Sie darauf, um dort fortzufahren, wo Sie aufgehört haben.</p>
<p><strong>Stapelverwaltung:</strong> Über die Aktionsschaltflächen lassen sich Stapel umbenennen, synchronisieren, zurücksetzen oder löschen (für die letzten beiden wird eine Bestätigung verlangt). Die FSRS-Parameter (Ziel-Retention, maximales Intervall, Streuung) können je Stapel in den Einstellungen (Zahnradsymbol) festgelegt werden.</p>
<p><strong>Retention: Ziel und Messung.</strong> Die <em>Ziel-Retention</em> ist Ihre Entscheidung über den Kompromiss zwischen Arbeitsaufwand und Erinnerungsqualität: Je höher sie ist, desto kürzer werden die Intervalle und desto mehr wiederholen Sie. Daneben zeigen die Einstellungen die <strong>gemessene Retention</strong> über Ihre eigenen Wiederholungen an — eine Information, niemals eine Steuerung: blunderDB ändert Ihr Ziel nicht, um Ihrer Erfolgsquote hinterherzulaufen. Unter etwa zwanzig Wiederholungen wird die Messung nicht angezeigt: Sie läse sich als Tatsache, obwohl sie nur Rauschen ist.</p>
<p>Eine Änderung der Retention <strong>wirkt nicht rückwirkend</strong>: Jede Karte übernimmt den neuen Rhythmus bei ihrer nächsten Wiederholung, und die bereits festgelegten Fälligkeiten verschieben sich nicht. Die Wirkung ist also allmählich und am selben Tag unsichtbar.</p>
<p>Das <em>maximale Intervall</em> begrenzt die Abstände. Ein neu angelegtes Deck beginnt bei einem Jahr: Eine Stellung, die der Algorithmus um mehrere Jahre verschieben würde, hat das Deck verlassen, ohne dass Sie es entschieden hätten, und Ihr eigenes Spiel ändert sich schneller als das. Ältere Decks behalten den Wert, den sie hatten.</p>
<h3>Micro-Trainings</h3>
<p>Das Anki-Panel lässt ein <strong>Urteil</strong> wiederholen; die Micro-Trainings üben die drei <strong>Berechnungen</strong>, die am Tisch unter Zeitdruck anfallen und die keine verteilte Wiederholung aufbaut. Der Befehl <code>train</code> startet eine Sitzung mit fünf Fragen:</p>
<ul>
<li><code>train pips</code> — die Pips des Spielers am Zug zählen, auf der gezeigten Stellung.</li>
<li><code>train epc</code> — den EPC desselben Spielers schätzen, auf einer Rennstellung, die die Engine auswerten kann.</li>
<li><code>train tp</code> — den Annahmepunkt eines langen Rennens bei einem zufällig gezogenen Stand nennen, den der Tabelle <code>tp2_live</code>.</li>
</ul>
<p>Die Frage IST die gezeigte Stellung: Das Brett ist das der Anwendung, und die Leiste darüber trägt nur die Frage, die Eingabe und die Korrektur. Die Antwort wird auf der Tastatur eingegeben und bestätigt (<em>Enter</em> prüft und geht dann weiter; <em>Esc</em> verlässt die Sitzung).</p>
<p>Die Toleranz hängt von der Übung ab und wird genannt statt erraten: Die Pip-Zählung hat <strong>keine</strong> — eine auf einen Pip genaue Addition ist eine falsche Addition — der EPC erlaubt einen halben Pip, der Annahmepunkt zwei Prozentpunkte. Am Ende zeigt die Sitzung die Zahl der richtigen Antworten und die <strong>mittlere</strong> Zeit je Frage.</p>
<p>Nur diese Zusammenfassung wird gespeichert, in den Metadaten der Datenbank: Die Sitzung hält keine Spur Frage für Frage fest, und nichts wird geschrieben, solange sie nicht beendet ist. Ein Abbruch auf halbem Weg speichert also nichts.</p>
<h4>Quiz: der Trainings-PR</h4>
<p><code>train quiz</code> stellt eine vierte Art von Frage. Das Anki-Panel lässt auswendig lernen; das Quiz <strong>prüft</strong>. Fünf bereits ausgewertete Stellungen werden aus der durchblätterten Liste gezogen, und es ist zu entscheiden:</p>
<ul>
<li>bei einer Zugentscheidung den Zug auf der Tastatur in Notation eingeben (<code>13/7 8/7</code>);</li>
<li>bei einer Verdopplungsentscheidung <em>Kein Doppel</em>, <em>Doppel, Annahme</em> oder <em>Doppel, Aufgabe</em> anklicken.</li>
</ul>
<p>Das Analyse-Panel bleibt verdeckt, solange die Frage keine Antwort hat: Es trägt die Antwort, und eine Frage, deren Antwort daneben steht, ist keine Frage.</p>
<p>Die Korrektur hält drei Ausgänge auseinander, und sie zu vermengen wäre gelogen. Ein <strong>unerlaubter Zug</strong> ist kein schlecht gewählter Zug — er ist ein Regelfehler. Ein <strong>erlaubter Zug, den die Engine nie bewertet hat</strong>, ist überhaupt kein Fehler: Er hat schlicht keinen Preis und kostet die Sitzung nichts. Ein bewerteter Zug kostet, was die Analyse sagt, in Millipunkten.</p>
<p>Am Ende zeigt die Sitzung einen <strong>Quiz-PR</strong>, berechnet mit der Formel, die die Statistiken auf das reale Spiel anwenden — 500 × mittlerer Fehler in normalisierter Equity. Das macht die beiden Zahlen vergleichbar: Ein Quiz-PR von 6 und ein Match-PR von 6 messen dasselbe auf derselben Skala.</p>
<h3>Metadaten-Panel</h3>
<p>Das Panel <strong>Metadaten</strong> zeigt die allgemeinen Informationen der aktuellen Datenbank an: Name, Beschreibung, Anzahl der Stellungen, Anzahl der Matches und Partien, Schemaversion. Erreichbar über den Befehl <code>meta</code>.</p>
<p>Es zeigt außerdem, <strong>sofern vorhanden</strong>, die Herkunft der Datenbank an — siehe Eine Datenbank weitergeben: Herkunft und Passwort. Bei einer gewöhnlichen Datenbank erscheint dieser Abschnitt nicht.</p>
<h3>Eine Datenbank weitergeben: Herkunft und Passwort</h3>
<p>Wer als Lehrender eine Positionsdatenbank verteilt, hat zwei voneinander unabhängige Mechanismen zur Hand, beide optional und beide <strong>beim Export</strong> zu wählen: die Datei mit ihrer Herkunft zu kennzeichnen und sie mit einem Passwort zu schützen.</p>
<div class="admonition note">
<p>Keiner von beiden verfolgt, was aus der Datei wird. blunderDB <strong>zeichnet auf der Empfängerseite nichts auf</strong>: Eine gekennzeichnete Datenbank zu öffnen ist genau wie jede andere zu öffnen, und nirgends wird festgehalten, wer sie wann geöffnet hat oder woher ihr Inhalt stammt.</p>
</div>
<h4>Eine Datenbank mit ihrer Herkunft kennzeichnen</h4>
<p>Das Exportfenster kommt mit einer einzigen Ansicht aus: dem Formular und einer Fortschrittsanzeige, die sich während des Schreibens darüberlegt. Es schließt sich nach Abschluss von selbst, und das Ergebnis erscheint in der Statusleiste.</p>
<p>Drei Punkte verdienen Beachtung:</p>
<ul>
<li><strong>Der Export umfasst die aktuell angezeigten Positionen</strong>, nicht die gesamte Datenbank. Nach einer Suche werden nur die Treffer exportiert — das Fenster weist oben darauf hin.</li>
<li><strong>Eine Sammlung, deren Positionen nicht alle in der Auswahl enthalten sind, kommt unvollständig an.</strong> Die Liste zeigt deshalb für jede Sammlung den abgedeckten Anteil („12/40“) und hebt ihn rot hervor, wenn er unvollständig ist.</li>
<li><strong>Turniere lassen sich nur zusammen mit den Partien exportieren</strong>: ohne sie gibt es die Verknüpfung Turnier–Partie nicht, und das Turnier käme leer an. Das Kästchen bleibt deaktiviert, solange „Partien einschließen“ nicht angehakt ist.</li>
</ul>
<p>Die Felder <em>Benutzer</em>, <em>Beschreibung</em> und <em>Datum</em> beschreiben die <strong>erzeugte Datei</strong>; sie sind aus der Quelldatenbank vorausgefüllt. Das Kästchen <em>Meine gespeicherten Filter</em> steht für sich: Es exportiert keine Inhalte, sondern Ihre eigenen gespeicherten Suchen, die in der Datenbank eines anderen nutzlos sind.</p>
<p>Ein Häkchen bei <strong>Diese Datei mit ihrer Herkunft kennzeichnen</strong> blendet zwei Felder ein:</p>
<ul>
<li><strong>Herkunft</strong> — was diese Datei ist und woher sie kommt, in Ihren eigenen Worten: „Unterricht von Jean Dupont — 12. März 2026“. Dieses Feld ist <strong>verpflichtend</strong>: Solange es leer ist, bleibt die Export-Schaltfläche inaktiv.</li>
<li><strong>Hinweis</strong>, optional — Nutzungsbedingungen, eine Kontaktadresse, die Bitte, die Datei nicht weiterzugeben.</li>
</ul>
<p>Die Kennzeichnung wird mit Ihrer Ausstelleridentität signiert. Sie ist damit <strong>unverfälschbar und nicht nachahmbar</strong>: Niemand kann sie verändern oder eine in Ihrem Namen erzeugen. Sie ist dagegen <strong>nicht unlöschbar</strong> — die verteilte Datei ist eine gewöhnliche SQLite-Datenbank, und blunderDB ist freie Software. Sie verhindert nichts: Sie sagt, woher die Datei kommt.</p>
<h4>Ausstelleridentität</h4>
<p>Die Kennzeichnungen werden mit Ihrer <strong>Ausstelleridentität</strong> signiert, die beim ersten Kennzeichnen einer Datei von selbst angelegt wird; es ist nichts einzurichten. Sie gehört zu einer Person und nicht zu einer Datenbank: Alle Ihre Dateien tragen denselben öffentlichen Fingerabdruck der Form <code>A3F1-9C24-7B05-E1D8</code>.</p>
<p>Sie können diesen Fingerabdruck Ihren Empfängern mitteilen, damit sie prüfen können, dass eine Datei tatsächlich von Ihnen stammt. Die Identität lässt sich als einzelne Datei (Endung <code>.bdbid</code>) von einem Rechner auf den anderen mitnehmen, auf Wunsch durch eine Passphrase geschützt. <strong>Mit dieser Datei kann in Ihrem Namen signiert werden: Geben Sie sie nicht weiter.</strong></p>
<p>In den Einstellungen (Zahnradsymbol in der Werkzeugleiste) zeigt die Registerkarte <em>Ausstelleridentität</em> Ihren Namen und Ihren Fingerabdruck an und bietet <em>Identität speichern…</em>, <em>Identität laden…</em> und <em>Neu erzeugen…</em> an.</p>
<div class="admonition warning">
<p><strong>Neu erzeugen widerruft nichts.</strong> Ein Wasserzeichen enthält den öffentlichen Schlüssel, mit dem es signiert wurde: Es lässt sich daher für immer aus sich selbst heraus prüfen. Ist Ihre Identitätsdatei abhandengekommen, kann derjenige, der sie besitzt, weiterhin unter Ihrem alten Fingerabdruck signieren, und diese Kennzeichnungen bleiben gültig.</p>
<p>Was Sie nach einem solchen Verlust schützt, ist keine Software: Es ist, Ihren neuen Fingerabdruck zu veröffentlichen und den alten gegenüber Ihren Empfängern für ungültig zu erklären.</p>
<p>Das Neuerzeugen überschreibt den aktuellen Schlüssel; blunderDB bietet an, ihn vor dem Ersetzen zu speichern.</p>
</div>
<h4>Eine Datenbank mit einem Passwort schützen</h4>
<p>Das Passwort wird verdeckt eingegeben, hier wie beim Öffnen einer geschützten Datei; das Augensymbol zeigt es an, <strong>solange es gedrückt gehalten wird</strong>, und verdeckt es wieder, sobald man loslässt.</p>
<p>Ein Häkchen bei <strong>Diese Datei mit einem Passwort schützen</strong> erzeugt eine Datei mit der Endung <code>.dbx</code> — auch dann, wenn Sie im Speichern-Dialog einen Namen auf <code>.db</code> gewählt hatten, denn dieser Dialog öffnet sich, bevor nach dem Passwort gefragt wird. Zum Öffnen verwenden Sie das gewohnte Öffnen einer Datenbank: Der Auswahldialog akzeptiert sowohl <code>.db</code> als auch <code>.dbx</code>. blunderDB fragt dann nach dem Passwort und legt daneben eine gewöhnliche Datenbank an; danach wird nichts mehr abgefragt.</p>
<p>Das Fenster bietet an, <strong>die geschützte Datei nach dem Öffnen zu löschen</strong>: Andernfalls behalten Sie denselben Inhalt unter zwei Namen. Das Kästchen ist standardmäßig nicht angehakt — die geschützte Datei bleibt Ihnen erhalten, wenn Sie sie weitergeben wollen — und gelöscht wird erst nach einem erfolgreichen Öffnen.</p>
<div class="admonition warning">
<p>Das Passwort schützt die Datei auf dem <strong>Transportweg</strong>, nicht die Datenbank. Es hindert Dritte daran, eine Datei zu öffnen, die in einem Download-Ordner liegt oder als versehentlich weitergeleiteter Anhang ankommt. Vor demjenigen, dem Sie das Passwort gegeben haben, schützt es nicht.</p>
</div>
<p>Das Passwort wird bei <strong>jedem</strong> Öffnen geprüft, auch wenn die Datei auf diesem Rechner schon einmal geöffnet wurde.</p>
<p>Technisch wird die Datenbank mit <strong>AES-256 im GCM-Modus</strong> verschlüsselt, mit einem Schlüssel, der per <strong>Argon2id</strong> (64 MiB Speicher, 3 Durchläufe, 4 Threads) aus dem Passwort abgeleitet wird, und einem zufälligen, für jede Datei eigenen Salt. Der GCM-Modus authentifiziert das Ganze: Ein falsches Passwort wird als solches erkannt, ebenso jede Veränderung der verschlüsselten Datei — man erhält nie stillschweigend eine beschädigte Datenbank.</p>
<p>Der Kopf der geschützten Datei bleibt <strong>unverschlüsselt</strong>: Ihre Herkunft ist auch ohne Passwort lesbar.</p>
<h4>Die Herkunft einer Datei lesen</h4>
<p>Öffnen Sie in der Anwendung die Datei und blenden Sie das Panel <strong>Metadaten</strong> ein (Befehl <code>meta</code>). Am Kopf des Panels erscheint schreibgeschützt ein Abschnitt <strong>Herkunft</strong>, der angibt, was eingetragen wurde, von wem, wann, und wie es um die Signatur steht:</p>
<ul>
<li>„✓ Signatur geprüft — von Ihnen gekennzeichnet“: Die Datei trägt Ihre Kennzeichnung, unversehrt;</li>
<li>„✓ Signatur geprüft“: Die Kennzeichnung ist unversehrt und stammt von einem anderen Schlüssel — vergleichen Sie ihren Fingerabdruck mit dem, den der Ersteller Ihnen mitgeteilt hat;</li>
<li>„⚠ Signatur ungültig“: Das Dokument wurde verändert oder gefälscht.</li>
</ul>
<p>Bei einer gewöhnlichen Datenbank erscheint dieser Abschnitt nicht.</p>
<p>Auf der Kommandozeile zeigt <code>blunderdb info --db datei.db</code> die Herkunft und den Zustand der Signatur an, <strong>ohne jemals in die Datei zu schreiben</strong>. Der Befehl funktioniert auch bei einer geschützten Datei, ohne das Passwort. Siehe <code>CLI_USAGE.md</code> für die Optionen <code>--watermark</code> und <code>--password</code> von <code>export</code> sowie für <code>identity</code> und <code>open</code>.</p>
<h4>Eine Datenbank für andere veröffentlichen</h4>
<p>Eine markierte Datenbank wird wie jede andere Datei verteilt — E-Mail, eigene Website, USB-Stick. blunderDB <strong>bietet keinen Dienst</strong>: kein Repository, keinen gehosteten Katalog, kein Konto. Das folgt unmittelbar aus seiner Bauart: auf der Seite dessen, der eine Datei erhält, wird nie etwas aufgezeichnet, es gäbe also nichts an einen Dienst zu melden, selbst wenn es einen gäbe.</p>
<p>Was eine veröffentlichte Datenbank für andere brauchbar macht, hängt an vier Feldern, die alle schon da sind:</p>
<ul>
<li><strong>Benutzer</strong> — wer sie zusammengestellt hat, unter dem Namen, der genannt werden soll.</li>
<li><strong>Beschreibung</strong> — was die Datenbank enthält, in einem Satz, der in eine Liste passt: „240 Dopplerentscheidungen beim Stand, kommentiert, mittleres Niveau“.</li>
<li><strong>Herkunft</strong> (des Wasserzeichens) — was diese Datei ist und für wen sie erzeugt wurde. Es ist das Erste, was der Empfänger im Panel <em>Metadaten</em> liest.</li>
<li><strong>Ausstellerfingerabdruck</strong> — veröffentlichen Sie ihn neben der Datei, nicht darin: an seinem Vergleich prüft der Empfänger, dass die Datei von Ihnen kommt und nicht von jemandem, der Ihren Namen übernommen hat.</li>
</ul>
<p>Eine ohne Wasserzeichen veröffentlichte Datenbank bleibt vollkommen brauchbar; sie ist einfach anonym, und das Panel <em>Metadaten</em> zeigt dann keinen Abschnitt <em>Herkunft</em>.</p>
<p>Um eine Datenbank bekannt zu machen, dient die Kategorie <em>Show and tell</em> der <code>Diskussionen des Repositorys &lt;https://github.com/kevung/blunderDB/discussions&gt;</code>_ als Verzeichnis: eine Liste, geführt von denen, die veröffentlichen, kein von blunderDB erbrachter Dienst. Eine dort anzukündigen braucht den Link, die vier obigen Felder und den Fingerabdruck.</p>
`,
    shortcuts: `
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
<td>Eine Datenbank in diese zusammenführen.</td>
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
<h3>Hilfe-Fenster</h3>
<table>
<thead>
<tr>
<th>Tastenkürzel</th>
<th>Aktion</th>
</tr>
</thead>
<tbody>
<tr>
<td>LINKS, h</td>
<td>Vorheriger Reiter.</td>
</tr>
<tr>
<td>RECHTS, l</td>
<td>Nächster Reiter.</td>
</tr>
<tr>
<td>OBEN, k</td>
<td>Nach oben scrollen.</td>
</tr>
<tr>
<td>UNTEN, j</td>
<td>Nach unten scrollen.</td>
</tr>
<tr>
<td>LEERTASTE</td>
<td>Nächste Seite.</td>
</tr>
<tr>
<td>Bild-auf</td>
<td>Anfang des Inhalts.</td>
</tr>
<tr>
<td>Bild-ab</td>
<td>Ende des Inhalts.</td>
</tr>
<tr>
<td>?, STRG-F, Esc</td>
<td>Hilfe schließen.</td>
</tr>
</tbody>
</table>
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
<td>Öffnet das Eval-Panel (Effective Pip Count, Gewinnwahrscheinlichkeit und Doppler-Urteil im Bearoff). <code>epc</code> ist der alte, weiterhin gültige Name dieses Panels.</td>
</tr>
<tr>
<td>met</td>
<td>Öffnet die Match-Equity-Tabelle Kazaross-XG2.</td>
</tr>
<tr>
<td>cm</td>
<td>Öffnet die Dopplerwürfel-Matrix: das Urteil der aktuellen Stellung bei jedem Punktestand eines Matches über 5, 7 oder 9 Punkte.</td>
</tr>
<tr>
<td>tags</td>
<td>Öffnet das Tag-Vokabular: die in dieser Datenbank verwendeten Tags mit der Zahl der Stellungen, anklickbar zum Starten der Suche.</td>
</tr>
<tr>
<td>log</td>
<td>Öffnet das Aktivitätsprotokoll: die letzten zweihundert Zeilen der Protokolldatei, mit dem Nötigen, um sie für einen Bericht zu kopieren oder den Ordner zu öffnen, der sie enthält.</td>
</tr>
<tr>
<td>ask</td>
<td>Übersetzt einen Satz in Worten — Französisch oder Englisch — in Suchtoken: <code>ask my cube blunders at a score</code>. Die Token werden in die Befehlszeile geschrieben, nicht ausgeführt: durchlesen, dann Enter. Was nicht verstanden wurde, wird gesagt, nie erraten.</td>
</tr>
<tr>
<td>like</td>
<td>Ersetzt die durchblätterte Liste durch die Stellungen, die der aktuellen am nächsten stehen — oder der, deren Index angegeben ist (<code>like 42</code>). Die Nähe ist eine Transportdistanz in Stein-Pips: Sie ist kein Filter, sie ordnet die ganze Datenbank, statt sie einzuschränken, und lässt sich daher nicht mit den Suchtoken kombinieren.</td>
</tr>
<tr>
<td>train</td>
<td>Startet eine Micro-Training-Sitzung. Nimmt ein Argument: <code>train pips</code> (Pip-Zählung), <code>train epc</code>, <code>train tp</code> (Annahmepunkt beim Matchstand), <code>train quiz</code> (der Zug oder die Verdopplungsentscheidung, bewertet gegen die gespeicherte Analyse). Fünf Fragen, auf Zeit, sofort korrigiert.</td>
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
<td>Importiert eine oder mehrere Stellungen/Matches aus einer Datei (xg, xgp, sgf, mat, txt, bgf). Mit einem Argument — <code>import XGID=…</code> oder <code>import OGID=…</code> — liest es die Kennung, statt eine Dateiauswahl zu öffnen, für den Fall, dass sie aus einer Nachricht, einem Forum oder einem Skript kommt.</td>
</tr>
<tr>
<td>delete, del, d</td>
<td>Löscht die aktuelle Stellung (mit Rückfrage); der Löschvorgang geht durch den Papierkorb und bleibt dreißig Tage rückgängig zu machen.</td>
</tr>
<tr>
<td>trash</td>
<td>Öffnet den Papierkorb: was gelöscht wurde, und womit es sich wiederherstellen lässt.</td>
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
<p>Diese Tabelle ist die Referenz der Suchgrammatik: die Befehlszeile, die Filterbibliothek und die Option <code>--query</code> von <code>blunderdb search</code> lesen alle dieselben Token. Die Spalte <em>CLI-Äquivalent</em> nennt, wenn es eine gibt, die Option von <code>search</code>, die dasselbe bewirkt (siehe Befehlszeilenschnittstelle (CLI)); ein Gedankenstrich zeigt einen Filter an, den nur die Grammatik ausdrückt.</p>
<p>Fünf Token tragen ihren Wert nicht selbst: Sie lesen ihn vom Suchbrett ab. <code>cube</code> und <code>score</code> übernehmen den dort eingestellten Doppler und Spielstand, <code>d</code> den Entscheidungstyp, <code>D</code> und <code>D1</code> die Würfel, <code>x</code> die im Reiter <em>Außer</em> gezeichnete Struktur. Ein Wurf steht also nie im Token selbst: <code>D65</code> gibt es nicht, nur die Ausschlussform trägt seine Ziffern (<code>xD65</code>). Auf der Kommandozeile, wo es kein Brett gibt, vergleichen diese Token mit einem leeren Brett; dort sind die Optionen der dritten Spalte zu verwenden.</p>
<p>Fehler und Equitys werden in <strong>Tausendstel Equity</strong> gezählt — den <em>Millipunkten</em> der Tabelle unten: <code>E&gt;100</code> behält Züge, die mindestens ein Zehntel Punkt gekostet haben, wobei ein Punkt 1000 Tausendsteln entspricht.</p>
<p>Zwei vollständige Suchen:</p>
<ul>
<li><code>s p&gt;30 w40,60 xco</code> — mehr als 30 Pips Rückstand, zwischen 40 % und 60 % Gewinnchance, kein Kommentar.</li>
<li><code>s ph:race E&gt;50 co:xg</code> — in der Rennphase, ein Zug, der mindestens 50 Tausendstel gekostet hat, und ein Kommentar aus eXtreme Gammon.</li>
</ul>
<table>
<thead>
<tr>
<th>Abfrage</th>
<th>Aktion</th>
<th>CLI-Äquivalent</th>
</tr>
</thead>
<tbody>
<tr>
<td>cube, cub, cu, c</td>
<td>Die Position erfüllt die Doppler-Konfiguration.</td>
<td><code>--cube</code></td>
</tr>
<tr>
<td>score, sco, sc, s</td>
<td>Die Position erfüllt den Spielstand.</td>
<td><code>--score1</code> <code>--score2</code></td>
</tr>
<tr>
<td>d</td>
<td>Die Position erfüllt den Entscheidungstyp (Stein oder Doppler).</td>
<td><code>--decision</code></td>
</tr>
<tr>
<td>D</td>
<td>Die Position erfüllt den Würfelwurf (beide Würfel, unabhängig von der Reihenfolge).</td>
<td><code>--dice 6,5</code></td>
</tr>
<tr>
<td>D1</td>
<td>Die Position erfüllt den Würfelwurf nur beim ersten Würfel (der Wert des ersten Würfels erscheint auf einem der beiden Würfel der Position).</td>
<td><code>--dice 6</code></td>
</tr>
<tr>
<td>xD65</td>
<td>Die Position wurde <strong>nicht</strong> mit dem Wurf 6-5 gespielt (unabhängig von der Reihenfolge). Der Wert wird im Token angezeigt; wiederholbar, um mehrere Würfe auszuschließen (<code>xD65 xD54</code>).</td>
<td>—</td>
</tr>
<tr>
<td>nc</td>
<td>Die Position ist kontaktlos.</td>
<td>—</td>
</tr>
<tr>
<td>ph:race</td>
<td>Die Stellung befindet sich in einer bestimmten Spielphase: <code>opening</code> (Eröffnung), <code>middlegame</code> (Mittelspiel), <code>race</code> (Wettlauf) oder <code>bearoff</code> (Auswürfeln). Wiederholbar (<code>ph:race ph:bearoff</code>). Die Kennzeichnung wird aus dem Brett abgeleitet und ist nie editierbar; <code>blunderdb repair</code> berechnet sie neu.</td>
<td><code>--phase</code></td>
</tr>
<tr>
<td>gt:holding</td>
<td>Die Stellung fällt unter einen bestimmten Spielplan, aus Sicht des Spielers am Zug: <code>race</code>, <code>bearin</code> (Einbringen unter Kontakt), <code>crunch</code>, <code>backgame</code>, <code>acepoint</code>, <code>blitz</code>, <code>primevprime</code>, <code>mutualholding</code>, <code>holding</code>, <code>contact</code>. Wiederholbar (<code>gt:holding gt:mutualholding</code>). Ein abgeleitetes Etikett wie die Phase: aus dem Brett berechnet, nie editierbar, von <code>blunderdb repair</code> neu berechnet.</td>
<td><code>--game-type</code></td>
</tr>
<tr>
<td>#prime</td>
<td>Die Stellung trägt dieses <strong>Tag</strong> in einem ihrer Kommentare. Ein Tag ist ein <code>#Wort</code> in der Prosa; nichts deklariert es. Der Vergleich ist abgegrenzt, also findet <code>#prime</code> nicht <code>#priming</code> — genau darin liegt der Unterschied zum Textfilter, der eine Teilzeichenkette sucht. Wiederholbar, und Tags <strong>addieren sich</strong> (<code>#prime #backgame</code> verlangt beide): eine Stellung trägt mehrere Tags, zwei zu nennen heißt also „beide“.</td>
<td>—</td>
</tr>
<tr>
<td>n&gt;x</td>
<td>Die Stellung kam in der Datenbank mehr als x-mal vor — die Zahl der Züge, die zu ihr führen, über alle Matches hinweg. Formen <code>n&gt;3</code>, <code>n&lt;2</code>, <code>n3,10</code> und <code>n4</code> (genau vier).</td>
<td>—</td>
</tr>
<tr>
<td>M</td>
<td>Die Position oder ihr Spiegelbild erfüllt die Filter.</td>
<td>—</td>
</tr>
<tr>
<td>i</td>
<td>Die Stellung wurde einzeln importiert und nicht durch einen Match-Import eingebracht.</td>
<td><code>--individual</code></td>
</tr>
<tr>
<td>fl</td>
<td>Die Position wurde in der Ursprungssoftware markiert, beim Import einer eXtreme-Gammon-Partie.</td>
<td><code>--flagged</code></td>
</tr>
<tr>
<td>x</td>
<td>Die Position enthält keinen Stein der Ausschlussstruktur (Registerkarte „Except“ des Suchpanels).</td>
<td>—</td>
</tr>
<tr>
<td>p&gt;x</td>
<td>Der Spieler liegt im Rennen mindestens x Pips zurück.</td>
<td><code>--pip-min</code></td>
</tr>
<tr>
<td>p&lt;x</td>
<td>Der Spieler liegt im Rennen höchstens x Pips zurück.</td>
<td><code>--pip-max</code></td>
</tr>
<tr>
<td>px,y</td>
<td>Der Spieler liegt im Rennen zwischen x und y Pips zurück.</td>
<td><code>--pip-min</code> <code>--pip-max</code></td>
</tr>
<tr>
<td>P&gt;x</td>
<td>Der Spieler hat ein Rennen von mindestens x Pips.</td>
<td>—</td>
</tr>
<tr>
<td>P&lt;x</td>
<td>Der Spieler hat ein Rennen von höchstens x Pips.</td>
<td>—</td>
</tr>
<tr>
<td>Px,y</td>
<td>Der Spieler hat ein Rennen zwischen x und y Pips.</td>
<td>—</td>
</tr>
<tr>
<td>e&gt;x</td>
<td>Die Equity (in Millipunkten) der Position ist größer als x.</td>
<td>—</td>
</tr>
<tr>
<td>e&lt;x</td>
<td>Die Equity (in Millipunkten) der Position ist kleiner als x.</td>
<td>—</td>
</tr>
<tr>
<td>ex,y</td>
<td>Die Equity (in Millipunkten) der Position liegt zwischen x und y.</td>
<td>—</td>
</tr>
<tr>
<td>E&gt;x</td>
<td>Der Fehler des von Spieler 1 gespielten Zuges (in Millipunkten) ist größer als x.</td>
<td><code>--move-error-min</code></td>
</tr>
<tr>
<td>E&lt;x</td>
<td>Der Fehler des von Spieler 1 gespielten Zuges (in Millipunkten) ist kleiner als x.</td>
<td><code>--move-error-max</code></td>
</tr>
<tr>
<td>Ex,y</td>
<td>Der Fehler des von Spieler 1 gespielten Zuges (in Millipunkten) liegt zwischen x und y.</td>
<td><code>--move-error-min</code> <code>--move-error-max</code></td>
</tr>
<tr>
<td>w&gt;x</td>
<td>Der Spieler hat Gewinnchancen von mehr als x %.</td>
<td><code>--winrate-min</code></td>
</tr>
<tr>
<td>w&lt;x</td>
<td>Der Spieler hat Gewinnchancen von weniger als x %.</td>
<td><code>--winrate-max</code></td>
</tr>
<tr>
<td>wx,y</td>
<td>Der Spieler hat Gewinnchancen zwischen x % und y %.</td>
<td><code>--winrate-min</code> <code>--winrate-max</code></td>
</tr>
<tr>
<td>g&gt;x</td>
<td>Der Spieler hat Gammon-Chancen von mehr als x %.</td>
<td>—</td>
</tr>
<tr>
<td>g&lt;x</td>
<td>Der Spieler hat Gammon-Chancen von weniger als x %.</td>
<td>—</td>
</tr>
<tr>
<td>gx,y</td>
<td>Der Spieler hat Gammon-Chancen zwischen x % und y %.</td>
<td>—</td>
</tr>
<tr>
<td>b&gt;x</td>
<td>Der Spieler hat Backgammon-Chancen von mehr als x %.</td>
<td>—</td>
</tr>
<tr>
<td>b&lt;x</td>
<td>Der Spieler hat Backgammon-Chancen von weniger als x %.</td>
<td>—</td>
</tr>
<tr>
<td>bx,y</td>
<td>Der Spieler hat Backgammon-Chancen zwischen x % und y %.</td>
<td>—</td>
</tr>
<tr>
<td>W&gt;x</td>
<td>Der Gegner hat Gewinnchancen von mehr als x %.</td>
<td>—</td>
</tr>
<tr>
<td>W&lt;x</td>
<td>Der Gegner hat Gewinnchancen von weniger als x %.</td>
<td>—</td>
</tr>
<tr>
<td>Wx,y</td>
<td>Der Gegner hat Gewinnchancen zwischen x % und y %.</td>
<td>—</td>
</tr>
<tr>
<td>G&gt;x</td>
<td>Der Gegner hat Gammon-Chancen von mehr als x %.</td>
<td>—</td>
</tr>
<tr>
<td>G&lt;x</td>
<td>Der Gegner hat Gammon-Chancen von weniger als x %.</td>
<td>—</td>
</tr>
<tr>
<td>Gx,y</td>
<td>Der Gegner hat Gammon-Chancen zwischen x % und y %.</td>
<td>—</td>
</tr>
<tr>
<td>B&gt;x</td>
<td>Der Gegner hat Backgammon-Chancen von mehr als x %.</td>
<td>—</td>
</tr>
<tr>
<td>B&lt;x</td>
<td>Der Gegner hat Backgammon-Chancen von weniger als x %.</td>
<td>—</td>
</tr>
<tr>
<td>Bx,y</td>
<td>Der Gegner hat Backgammon-Chancen zwischen x % und y %.</td>
<td>—</td>
</tr>
<tr>
<td>o&gt;x</td>
<td>Der Spieler hat mindestens x ausgewürfelte Steine.</td>
<td><code>--off1-min</code></td>
</tr>
<tr>
<td>o&lt;x</td>
<td>Der Spieler hat höchstens x ausgewürfelte Steine.</td>
<td>—</td>
</tr>
<tr>
<td>ox,y</td>
<td>Der Spieler hat zwischen x und y ausgewürfelte Steine.</td>
<td>—</td>
</tr>
<tr>
<td>O&gt;x</td>
<td>Der Gegner hat mindestens x ausgewürfelte Steine.</td>
<td><code>--off2-min</code></td>
</tr>
<tr>
<td>O&lt;x</td>
<td>Der Gegner hat höchstens x ausgewürfelte Steine.</td>
<td>—</td>
</tr>
<tr>
<td>Ox,y</td>
<td>Der Gegner hat zwischen x und y ausgewürfelte Steine.</td>
<td>—</td>
</tr>
<tr>
<td>k&gt;x</td>
<td>Der Spieler hat mindestens x rückständige Steine.</td>
<td>—</td>
</tr>
<tr>
<td>k&lt;x</td>
<td>Der Spieler hat höchstens x rückständige Steine.</td>
<td>—</td>
</tr>
<tr>
<td>kx,y</td>
<td>Der Spieler hat zwischen x und y rückständige Steine.</td>
<td>—</td>
</tr>
<tr>
<td>K&gt;x</td>
<td>Der Gegner hat mindestens x rückständige Steine.</td>
<td>—</td>
</tr>
<tr>
<td>K&lt;x</td>
<td>Der Gegner hat höchstens x rückständige Steine.</td>
<td>—</td>
</tr>
<tr>
<td>Kx,y</td>
<td>Der Gegner hat zwischen x und y rückständige Steine.</td>
<td>—</td>
</tr>
<tr>
<td>z&gt;x</td>
<td>Der Spieler hat mindestens x Steine in der Zone.</td>
<td>—</td>
</tr>
<tr>
<td>z&lt;x</td>
<td>Der Spieler hat höchstens x Steine in der Zone.</td>
<td>—</td>
</tr>
<tr>
<td>zx,y</td>
<td>Der Spieler hat zwischen x und y Steine in der Zone.</td>
<td>—</td>
</tr>
<tr>
<td>Z&gt;x</td>
<td>Der Gegner hat mindestens x Steine in der Zone.</td>
<td>—</td>
</tr>
<tr>
<td>Z&lt;x</td>
<td>Der Gegner hat höchstens x Steine in der Zone.</td>
<td>—</td>
</tr>
<tr>
<td>Zx,y</td>
<td>Der Gegner hat zwischen x und y Steine in der Zone.</td>
<td>—</td>
</tr>
<tr>
<td>bo&gt;x</td>
<td>Der Spieler hat mindestens x Blots im Outfield.</td>
<td>—</td>
</tr>
<tr>
<td>bo&lt;x</td>
<td>Der Spieler hat höchstens x Blots im Outfield.</td>
<td>—</td>
</tr>
<tr>
<td>box,y</td>
<td>Der Spieler hat zwischen x und y Blots im Outfield.</td>
<td>—</td>
</tr>
<tr>
<td>BO&gt;x</td>
<td>Der Gegner hat mindestens x Blots im Outfield.</td>
<td>—</td>
</tr>
<tr>
<td>BO&lt;x</td>
<td>Der Gegner hat höchstens x Blots im Outfield.</td>
<td>—</td>
</tr>
<tr>
<td>BOx,y</td>
<td>Der Gegner hat zwischen x und y Blots im Outfield.</td>
<td>—</td>
</tr>
<tr>
<td>bj&gt;x</td>
<td>Der Spieler hat mindestens x Blots im Heimfeld.</td>
<td>—</td>
</tr>
<tr>
<td>bj&lt;x</td>
<td>Der Spieler hat höchstens x Blots im Heimfeld.</td>
<td>—</td>
</tr>
<tr>
<td>bjx,y</td>
<td>Der Spieler hat zwischen x und y Blots im Heimfeld.</td>
<td>—</td>
</tr>
<tr>
<td>BJ&gt;x</td>
<td>Der Gegner hat mindestens x Blots im Heimfeld.</td>
<td>—</td>
</tr>
<tr>
<td>BJ&lt;x</td>
<td>Der Gegner hat höchstens x Blots im Heimfeld.</td>
<td>—</td>
</tr>
<tr>
<td>BJx,y</td>
<td>Der Gegner hat zwischen x und y Blots im Heimfeld.</td>
<td>—</td>
</tr>
<tr>
<td><code>t'wort1;wort2;...'</code></td>
<td>Die Kommentare der Position enthalten mindestens eines der Wörter.</td>
<td>—</td>
</tr>
<tr>
<td>co</td>
<td>Die Position hat einen Kommentar, unabhängig vom Inhalt.</td>
<td><code>--has-comment</code></td>
</tr>
<tr>
<td>xco</td>
<td>Die Position hat keinen Kommentar.</td>
<td><code>--no-comment</code></td>
</tr>
<tr>
<td>co:user</td>
<td>Die Stellung trägt einen Kommentar einer bestimmten Herkunft: <code>user</code> (von Ihnen geschrieben), <code>xg</code>, <code>gnubg</code>, <code>bgf</code> (durch einen Partie-Import mitgebracht) oder <code>unknown</code>. Wiederholbar (<code>co:xg co:gnubg</code>).</td>
<td><code>--comment-origin</code></td>
</tr>
<tr>
<td><code>m'muster1,muster2,...'</code></td>
<td>Die besten Steinzüge, die mindestens eines der Muster enthalten.</td>
<td>—</td>
</tr>
<tr>
<td><code>m'ND,DT,DP,...'</code></td>
<td>Die besten Doppler-Entscheidungen für No Double/Take, Double Take, Double Pass.</td>
<td>—</td>
</tr>
<tr>
<td>T&gt;x</td>
<td>Datum des Hinzufügens der Position nach x (JJJJ/MM/TT).</td>
<td>—</td>
</tr>
<tr>
<td>T&lt;x</td>
<td>Datum des Hinzufügens der Position vor x (JJJJ/MM/TT).</td>
<td>—</td>
</tr>
<tr>
<td>Tx,y</td>
<td>Datum des Hinzufügens der Position zwischen x und y (JJJJ/MM/TT).</td>
<td>—</td>
</tr>
<tr>
<td>max</td>
<td>Sucht im Match mit der ID x (z. B. ma3).</td>
<td><code>--match-ids</code></td>
</tr>
<tr>
<td>max,y</td>
<td>Sucht in den Matches mit den IDs von x bis y (z. B. ma2,5).</td>
<td><code>--match-ids</code></td>
</tr>
<tr>
<td>tnx</td>
<td>Sucht im Turnier mit der ID x (z. B. tn1).</td>
<td><code>--tournament-ids</code></td>
</tr>
<tr>
<td>tnx,y</td>
<td>Sucht in den Turnieren mit den IDs von x bis y (z. B. tn1,3).</td>
<td><code>--tournament-ids</code></td>
</tr>
<tr>
<td>idx</td>
<td>Die Position mit der Kennung x suchen (z. B. id12).</td>
<td><code>--position-ids</code></td>
</tr>
<tr>
<td>idx,y</td>
<td>Die Positionen mit den Kennungen x bis y suchen (z. B. id5,10).</td>
<td><code>--position-ids</code></td>
</tr>
<tr>
<td><code>pl'Name'</code></td>
<td>Stellungen aus einer Partie suchen, an der der genannte Spieler an einer der beiden Seiten beteiligt war (z. B. <code>pl'Alice'</code>). Groß-/Kleinschreibung wird ignoriert.</td>
<td>—</td>
</tr>
</tbody>
</table>
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
`,
    about: `
<h3>Version</h3>
<p>Application version: {appVersion}</p>
<p>Database version: {dbVersion}</p>
<p>
    <a href="https://kevung.github.io/blunderDB/de/" target="_blank" rel="noopener noreferrer">Online-Dokumentation</a> ·
    <a href="https://kevung.github.io/blunderDB/de/historique.html" target="_blank" rel="noopener noreferrer">Versionshistorie</a>
</p>

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
    <li>Matchdateien werden von <em>xgparser</em>, <em>gnubgparser</em> und <em>bgfparser</em> (MIT) gelesen.</li>
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
