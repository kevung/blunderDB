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
<h3>Johdanto</h3>
<p>
    blunderDB on ohjelmisto backgammon-asematietokantojen luomiseen. Sen tärkein vahvuus on tarjota yksi paikka, johon koota asemat, joita pelaaja on kohdannut (verkossa, turnauksissa), ja
    mahdollistaa näiden asemien uudelleentutkiminen suodattamalla niitä erilaisilla mielivaltaisesti yhdisteltävillä suodattimilla. blunderDB:tä voi käyttää myös viiteasemien luettelojen luomiseen.
</p>
<p>Asemat tallennetaan tietokantaan, jota edustaa .db-tiedosto.</p>

<h3>Päätoiminnot</h3>
<p>blunderDB:n tärkeimmät mahdolliset toiminnot ovat:</p>
<ul>
    <li>uuden aseman lisääminen,</li>
    <li>olemassa olevan aseman muokkaaminen,</li>
    <li>laudan kopioiminen PNG-kuvana leikepöydälle (<strong>Ctrl+X</strong>) tai laudan ja sen analyysin kopioiminen (<strong>Ctrl+X, Ctrl+X</strong>),</li>
    <li>olemassa olevan aseman poistaminen,</li>
    <li>yhden tai useamman aseman hakeminen,</li>
    <li>ottelujen tuominen eri lähteistä (XG, GNUbg, BGBlitz, Jellyfish), mukaan lukien kommentit XG-tiedostoista,</li>
    <li>tuodun ottelun siirtojen selaaminen,</li>
    <li>asemien järjestäminen kokoelmiin,</li>
    <li>ottelujen järjestäminen turnauksiin,</li>
    <li>erän analysointi komentoriviltä analyysittä olevista asemista sisäänrakennetun gammonNet-arvioijan avulla (blunderDB:n komento <strong>analyze</strong>).</li>
</ul>
<p>Käyttäjä voi vapaasti merkitä asemia tunnisteilla ja varustaa ne kommenteilla.</p>

<h3>Käyttöliittymän kuvaus</h3>
<p>blunderDB:n käyttöliittymä on rakenteeltaan ylhäältä alas seuraava:</p>
<ul>
    <li>[ylhäällä] työkalupalkki, joka kokoaa kaikki tietokantaan kohdistuvat päätoiminnot,</li>
    <li>[keskellä] päänäyttöalue, jolla voi näyttää tai muokata backgammon-asemia,</li>
    <li>[alhaalla] tilapalkki, joka sisältää komentorivin ja esittää erilaista tietoa nykyisestä asemasta.</li>
</ul>
<p>Voit avata paneeleja, jotka:</p>
<ul>
    <li>näyttävät nykyiseen asemaan liittyvät analyysitiedot (XG:stä, GNUbg:stä tai BGBlitzistä),</li>
    <li>näyttävät, lisäävät tai muokkaavat kommentteja,</li>
    <li>selaavat tuotuja otteluja ja navigoivat niiden siirtojen läpi (Ottelu-paneeli),</li>
    <li>hallitsevat asemakokoelmia (Kokoelma-paneeli),</li>
    <li>tutkivat asemia välitoistolla (Anki-paneeli),</li>
    <li>hallitsevat turnauksia (Turnaus-paneeli),</li>
    <li>näyttävät suoritustilastoja (Tilasto-paneeli),</li>
    <li>arvioivat minkä tahansa aseman sisäänrakennetulla moottorilla ja laskevat bearoff-aseman EPC:n (Eval-paneeli),</li>
    <li>selaavat tallennettuja hakusuodattimia (Suodatinkirjasto-paneeli),</li>
    <li>selaavat hakuhistoriaa (Hakuhistoria-paneeli).</li>
</ul>
<p>Päänäyttöalue tarjoaa käyttäjälle:</p>
<ul>
    <li>laudan backgammon-aseman näyttämiseen tai muokkaamiseen,</li>
    <li>kuution tason ja omistajan,</li>
    <li>kummankin pelaajan pip-luvun,</li>
    <li>kummankin pelaajan pistetilanteen,</li>
    <li>pelattavat nopat. Jos nopissa ei näy arvoa, noppien sijainti osoittaa, kummalla pelaajalla on vuoro ja että asema on kuutiopäätös.</li>
</ul>
<p>Tilapalkki näyttää vasemmalta oikealle:</p>
<ul>
    <li>komentorivin (avaa painamalla <strong>välilyönti</strong>),</li>
    <li>viimeiseen suoritettuun toimintoon liittyvän ilmoituksen,</li>
    <li>nykyisen aseman indeksin, jota seuraa asemien kokonaismäärä (tai siirto-/pelitiedot otteluja selattaessa).</li>
</ul>
<p>Käyttäjän hausta tulleiden asemien tapauksessa tilapalkissa ilmoitettu asemien määrä vastaa suodatettujen asemien määrää.</p>

<h3>Asemien selaaminen</h3>
<p>Oletuksena blunderDB mahdollistaa:</p>
<ul>
    <li>nykyisen kirjaston eri asemien selaamisen,</li>
    <li>asemaan liittyvien analyysitietojen näyttämisen,</li>
    <li>asemaa koskevien kommenttien näyttämisen, lisäämisen ja muokkaamisen.</li>
</ul>

<h3>Asemien muokkaaminen</h3>
<p>
    <strong>Tab</strong>-näppäimen painaminen avaa hakupaneelin ja mahdollistaa aseman muokkaamisen laudalla sen lisäämiseksi tietokantaan tai haettavan asemarakenteen määrittämiseksi. Nappuloiden
    jakauman, kuution, pistetilanteen ja vuoron voi muokata hiirellä.
</p>

<h3>Komentorivi</h3>
<p>
    Tilapalkkiin sisältyvä komentorivi mahdollistaa kaikkien blunderDB:n toimintojen suorittamisen: tietokantatoiminnot, asemanavigoinnin, analyysin ja kommenttien näyttämisen, asemien hakemisen
    suodattimilla... Käyttöliittymään tutustumisen jälkeen on suositeltavaa siirtyä vähitellen käyttämään komentoriviä, joka mahdollistaa tehokkaan ja sujuvan blunderDB:n käytön, erityisesti
    asemahakutoiminnoissa.
</p>
<p>
    Avaa komentorivi painamalla <strong>välilyönti</strong>-näppäintä. Tilapalkkiin ilmestyy kehote. Kirjoita komentosi ja suorita se painamalla <strong>Enter</strong>. Peruuta painamalla
    <strong>Escape</strong>.
</p>
<p>
    blunderDB suorittaa käyttäjän lähettämät kyselyt edellyttäen, että ne ovat kelvollisia, ja muuttaa tarvittaessa välittömästi tietokannan tilaa. Käyttäjältä ei vaadita erillisiä
    tallennustoimintoja.
</p>
<p>
    Tarkentaaksesi hakua aiemmin suodatettujen asemien sisällä käytä <strong>ss</strong>-komentoa, jota seuraavat suodattimet (esim. <strong>ss nc</strong>). Tämä rajaa haun vain parhaillaan
    näytettäviin asemiin, mikä mahdollistaa tulosten asteittaisen kaventamisen. Hakupaneeli (<strong>Ctrl+F</strong>) tarjoaa myös "Hae nykyisistä tuloksista" -valintaruudun samaa toiminnallisuutta
    varten.
</p>

<h3>Eval-paneeli</h3>
<p>
    <strong>Eval</strong>-paneeli arvioi minkä tahansa laudalla olevan aseman: voitto-, gammon- ja backgammon-todennäköisyydet, equityn, järjestetyt siirtoehdokkaat ja sen yhden ratkaisun, jota asema
    vaatii — siirron pelaamisen tai tuplaamisen. Laskennan tekee sisäänrakennettu gammonNet: eXtreme Gammonia tai GNU Backgammonia ei tarvita.
</p>
<p>
    Avaa se painamalla <strong>Ctrl+E</strong>, napsauttamalla alapaneelin Eval-välilehteä tai kirjoittamalla <strong>epc</strong> komentoriville. Lauta avautuu tavalliseen ulosmenoasetelmaan (15
    nappulaa), ellei sinne ole lähetetty asemaa tietokannasta. Nappuloita lisätään ja poistetaan vapaasti hiirellä; arvio seuraa jokaista muutosta.
</p>
<p>Ulosmenoasemassa paneeli <strong>erikoistuu</strong>: toinen taulukko, pelaajittain, näyttää EPC:n (Effective Pip Count) laskettuna GNUbg:n yksipuolisesta 6 pisteen ulosmenotietokannasta —</p>
<ul>
    <li><strong>EPC</strong>: keskimääräinen pip-määrä kaikkien nappuloiden ulosmenoon,</li>
    <li><strong>Pip Count</strong>: raaka pip-luku,</li>
    <li><strong>Wastage</strong>: EPC:n ja pip-luvun erotus,</li>
    <li><strong>Avg Rolls</strong>: keskimääräinen heittojen määrä kaikkien nappuloiden ulosmenoon,</li>
    <li><strong>Std Dev</strong>: tuon heittomäärän keskihajonta.</li>
</ul>
<p>Kun molemmilla pelaajilla on nappuloita kotialueellaan, vertailuosio näyttää EPC- ja pip-erot.</p>
<p>
    Puhtaassa juoksussa lisätaulukko näyttää molempien pelaajien voittotodennäköisyydet ja, kun asema kuuluu kaksipuolisen tietokannan piiriin (6 nappulan taulukko pelaajaa kohden lasketaan
    ensimmäisellä käynnistyksellä, 11 nappulan laajennettu taulukko lasketaan asetusten Bearoff-välilehdeltä), tarkat money-equityt ja parhaan tuplausratkaisun. Tämän alueen ulkopuolella
    voittotodennäköisyys on arvio (merkintä ”arvioitu” virhemarginaaleineen) eikä ratkaisua näytetä. Vuorossa oleva pelaaja vaihdetaan napsauttamalla pelaajan ulostulo-/pisteruutua ja tuplauskuution
    asema napsauttamalla laudan kuutiota.
</p>
<p>
    <strong>Haaste</strong>-valintaruutu piilottaa tulokset jokaisen aseman muutoksen yhteydessä; napsauta aluetta paljastaaksesi sen — mainio tapa harjoitella equityn, EPC:n tai tuplausratkaisun
    arviointia ennen tarkistusta.
</p>
<p>Sulje Eval-paneeli painamalla <strong>Ctrl+E</strong> uudelleen tai vaihtamalla toiseen välilehteen.</p>

<h3>Ottelunavigointi</h3>
<p>
    blunderDB mahdollistaa tuotujen ottelujen siirtojen selaamisen. Avaa Ottelu-paneeli painamalla <strong>Ctrl+Tab</strong> ja kaksoisnapsauta ottelua (tai paina <strong>Enter</strong>) ladataksesi
    sen asemat.
</p>
<p>
    Ottelua selattaessa viimeksi vierailtu asema tallennetaan ja palautetaan automaattisesti. Käytä <strong>vasen</strong>/<strong>oikea</strong>-näppäimiä asemien välillä liikkumiseen ja
    <strong>PageUp</strong>/<strong>PageDown</strong> pelien välillä hyppäämiseen.
</p>
<p>Analyysipaneeli (<strong>Ctrl+L</strong>) näyttää kunkin siirron analyysin, jossa pelattu siirto on korostettu. Paina <strong>d</strong> vaihtaaksesi nappula- ja kuutioanalyysin välillä.</p>

<h3>Kokoelmat</h3>
<p>
    Kokoelmat mahdollistavat asemien järjestämisen mukautettuihin ryhmiin. Avaa Kokoelma-paneeli painamalla <strong>Ctrl+B</strong> ja kaksoisnapsauta sitten kokoelmaa selataksesi sen asemia.
    Kokoelmat ja niiden sisältämät asemat voi järjestää uudelleen vetämällä ja pudottamalla.
</p>

<h3>Anki (välitoisto)</h3>
<p>Anki-paneeli (<strong>Ctrl+K</strong>) tarjoaa välitoiston backgammon-asemien tutkimiseen FSRS-algoritmilla.</p>
<p>
    <strong>Pakkojen luominen:</strong> Napsauta <em>Uusi pakka</em> luodaksesi pakan kokoelmasta tai nykyisistä hakutuloksista. Hakuun perustuvat pakat synkronoituvat automaattisesti, kun Anki-
    välilehti aktivoidaan.
</p>
<p>
    <strong>Kertaaminen:</strong> Valitse pakka ja napsauta <em>Opiskele</em> (tai kaksoisnapsauta pakkaa) aloittaaksesi erääntyneiden korttien kertaamisen. Jokainen kortti näyttää vastaavan aseman
    laudalla. Arvioi muistisi näppäimillä <strong>1</strong> (Uudelleen), <strong>2</strong> (Vaikea), <strong>3</strong> (Hyvä) tai <strong>4</strong> (Helppo). Paina
    <strong>Esc</strong> lopettaaksesi ja palataksesi pakkalistaan.
</p>
<p>
    <strong>Istunnon rajoittaminen:</strong> Pakan asetuksissa voit rajata istunnon korttimäärään. Istunto päättyy silloin ilmoittaen syyn, ja vapaa harjoittelu on yhä käytettävissä ilman vaikutusta
    aikatauluun. Raja <em>0</em> ei tarjoa yhtään korttia — se ei ole sama asia kuin rajaton.
</p>
<p>
    <strong>Muistiinjäänti:</strong> Tavoitetaso on sinun valintasi työmäärän ja laadun välillä. Asetukset näyttävät sen vieressä omista kertauksistasi <em>mitatun</em> tason — tieto, ei ohjaus.
    Tavoitteen muutos ei ole takautuva: kukin kortti omaksuu uuden rytmin seuraavassa kertauksessaan.
</p>
<p>
    <strong>Vastauksen näyttäminen:</strong> Kortti esittää kysymyksen; mieti se ensin ja paina sitten <strong>välilyöntiä</strong> (tai napsauta peitettyä aluetta) paljastaaksesi aseman tallennetun
    analyysin. Se ilmestyy arviointipainikkeiden alle, jotka pysyvät ulottuvilla. Vastausta ei tarvitse paljastaa arvioidakseen, ja se peittyy taas seuraavan kortin kohdalla — ei kuitenkaan pelkästä
    välilehden vaihdosta.
</p>
<p>
    <strong>Pysäytä/jatka:</strong> Voit pysäyttää kertausistunnon milloin tahansa painamalla <strong>Esc</strong>. Painike muuttuu muotoon <em>Jatka</em> ja näyttää edistymisesi. Napsauta sitä
    jatkaaksesi siitä, mihin jäit.
</p>
<p>
    <strong>Pakkojen hallinta:</strong> Käytä toimintopainikkeita pakkojen nimeämiseen uudelleen, synkronointiin, nollaamiseen tai poistamiseen. FSRS-parametrit (muistitavoite, maksimiväli, fuzz)
    voidaan määrittää pakkakohtaisesti Asetuksissa (ratas-kuvake).
</p>

<h3>Turnaukset</h3>
<p>
    Turnaukset mahdollistavat ottelujen ryhmittelyn tapahtuman mukaan. Tuonnissa ottelu sijoitetaan turnaukseen, jonka sen tiedosto nimeää ja joka luodaan tarvittaessa; jo sijoitettua ottelua ei
    koskaan siirretä. Avaa Turnaus-paneeli painamalla <strong>Ctrl+Y</strong> hallitaksesi turnauksia ja liittääksesi otteluita niihin.
</p>

<h3>Tilastot</h3>
<p>
    Tilasto-paneeli (<strong>Ctrl+D</strong>) näyttää suoritustilastot (PR ja MWC-kustannus), jotka lasketaan kaikista tuoduista asemista. Käytä suodatinpalkkia rajataksesi analyysiä pelaajan,
    turnauksen, päivämääräalueen, päätöstyypin tai ottelun pituuden mukaan. Napsauta mitä tahansa mittaria porautuaksesi vastaaviin asemiin. <strong>Pelaajat</strong>-välilehti listaa
    pelaajakohtaisesti otteluiden määrän, tuloksen, päätökset, PR:n (nappulat ja tuplauskuutio), Snowien, blundersit ja tunnettujen heittojen perusteella mitatun tuurin.
</p>

<h3>Vesileima ja suojattu vienti</h3>
<p>Viennin yhteydessä (<strong>export_db</strong> tai Vie-valintaikkuna) voi vapaasti ottaa käyttöön kaksi toisistaan riippumatonta suojausta, jommankumman tai molemmat yhdessä:</p>
<ul>
    <li>
        <strong>Vesileima:</strong> merkitsee vietyyn tiedostoon sen alkuperän (kuka sen tuotti, valinnainen huomautus). Vesileima on allekirjoitettu merkitsijän identiteetilläsi: sitä ei voi muuttaa
        eikä väärentää jonkun toisen nimissä — mutta se ei ole poistamaton eikä estä yhtäkään kopiota.
    </li>
    <li>
        <strong>Salasana:</strong> asettaa viennin salattuun <strong>.dbx</strong>-säiliöön. Se suojaa tiedoston siirron ajaksi, ei itse tietokantaa — se, jolle annat salasanan, voi avata sen — ja
        alkuperä näkyy ilman salasanaakin.
    </li>
</ul>
<p>
    Merkitsijän identiteettisi, avain joka allekirjoittaa vesileimasi, luodaan automaattisesti ensimmäisen kerran, kun vienti merkitään alkuperällään. Tarkastele, vie tai luo se uudelleen asetusten
    välilehdeltä <strong>Merkitsijän identiteetti</strong>.
</p>
`,
    shortcuts: `
<h3>Tietokanta</h3>
<table>
<thead>
<tr>
<th>Oikotie</th>
<th>Toiminto</th>
</tr>
</thead>
<tbody>
<tr>
<td>CTRL-N</td>
<td>Luo uusi tietokanta.</td>
</tr>
<tr>
<td>CTRL-O</td>
<td>Avaa olemassa oleva tietokanta.</td>
</tr>
<tr>
<td>CTRL-SHIFT-I</td>
<td>Yhdistä tietokanta tähän.</td>
</tr>
<tr>
<td>CTRL-SHIFT-S</td>
<td>Vie tietokanta.</td>
</tr>
<tr>
<td>CTRL-Q</td>
<td>Sulje blunderDB.</td>
</tr>
<tr>
<td>CTRL-M</td>
<td>Muokkaa tietokannan metatietoja.</td>
</tr>
</tbody>
</table>
<h3>Asema</h3>
<table>
<thead>
<tr>
<th>Oikotie</th>
<th>Toiminto</th>
</tr>
</thead>
<tbody>
<tr>
<td>CTRL-I</td>
<td>Tuo yksi tai useampi asema/ottelu tiedostosta (xg, xgp, sgf, mat, txt, bgf).</td>
</tr>
<tr>
<td>CTRL-SHIFT-F</td>
<td>Tuo ottelu-/asematiedostojen kansio rekursiivisesti.</td>
</tr>
<tr>
<td>CTRL-C</td>
<td>Kopioi asema leikepöydälle.</td>
</tr>
<tr>
<td>CTRL-X</td>
<td>Kopioi laudan kuva leikepöydälle (PNG).</td>
</tr>
<tr>
<td>CTRL-X CTRL-X</td>
<td>Kopioi laudan ja analyysin kuva leikepöydälle (PNG).</td>
</tr>
<tr>
<td>CTRL-V</td>
<td>Liitä asema leikepöydältä (muoto tunnistetaan automaattisesti).</td>
</tr>
<tr>
<td>CTRL-S</td>
<td>Tallenna asema.</td>
</tr>
<tr>
<td>CTRL-U</td>
<td>Päivitä asema.</td>
</tr>
<tr>
<td>Del</td>
<td>Poista nykyinen asema (vahvistus pyydetään).</td>
</tr>
<tr>
<td>ASKELPALAUTIN</td>
<td>Nollaa lauta, tuplauskuutio, tulos ja nopat.</td>
</tr>
<tr>
<td>CTRL-G</td>
<td>Näytä aseman metatiedot.</td>
</tr>
</tbody>
</table>
<h3>Navigointi</h3>
<table>
<thead>
<tr>
<th>Oikotie</th>
<th>Toiminto</th>
</tr>
</thead>
<tbody>
<tr>
<td>CTRL-R</td>
<td>Lataa kaikki tietokannan asemat uudelleen.</td>
</tr>
<tr>
<td>PageUp, h</td>
<td>Ensimmäinen asema / Edellinen peli (ottelunavigointi).</td>
</tr>
<tr>
<td>VASEN, k</td>
<td>Edellinen asema.</td>
</tr>
<tr>
<td>OIKEA, j</td>
<td>Seuraava asema.</td>
</tr>
<tr>
<td>YLÖS, k</td>
<td>Edellinen siirto (kun analyysissa on valittu siirto).</td>
</tr>
<tr>
<td>ALAS, j</td>
<td>Seuraava siirto (kun analyysissa on valittu siirto).</td>
</tr>
<tr>
<td>PageDown, l</td>
<td>Viimeinen asema / Seuraava peli (ottelunavigointi).</td>
</tr>
<tr>
<td>r</td>
<td>Lataa satunnainen asema.</td>
</tr>
</tbody>
</table>
<h3>Näyttö</h3>
<table>
<thead>
<tr>
<th>Oikotie</th>
<th>Toiminto</th>
</tr>
</thead>
<tbody>
<tr>
<td>CTRL-VASEN</td>
<td>Laudan suunta vasemmalle.</td>
</tr>
<tr>
<td>CTRL-OIKEA</td>
<td>Laudan suunta oikealle.</td>
</tr>
<tr>
<td>p</td>
<td>Näytä/piilota pip-laskuri.</td>
</tr>
</tbody>
</table>
<h3>Toiminnot</h3>
<table>
<thead>
<tr>
<th>Oikotie</th>
<th>Toiminto</th>
</tr>
</thead>
<tbody>
<tr>
<td>TAB</td>
<td>Avaa hakupaneeli (aseman editori).</td>
</tr>
<tr>
<td>VÄLILYÖNTI</td>
<td>Avaa komentorivi.</td>
</tr>
</tbody>
</table>
<h3>Työkalut</h3>
<table>
<thead>
<tr>
<th>Oikotie</th>
<th>Toiminto</th>
</tr>
</thead>
<tbody>
<tr>
<td>CTRL-L</td>
<td>Näytä/piilota analyysi.</td>
</tr>
<tr>
<td>CTRL-P</td>
<td>Näytä/piilota kommentit.</td>
</tr>
<tr>
<td>CTRL-K</td>
<td>Näytä/piilota Anki-paneeli (välitoistoharjoittelu).</td>
</tr>
<tr>
<td>CTRL-F</td>
<td>Näytä/piilota hakupaneeli.</td>
</tr>
<tr>
<td>CTRL-Tab</td>
<td>Näytä/piilota ottelupaneeli.</td>
</tr>
<tr>
<td>CTRL-B</td>
<td>Näytä/piilota kokoelmapaneeli.</td>
</tr>
<tr>
<td>CTRL-Y</td>
<td>Näytä/piilota turnauspaneeli.</td>
</tr>
<tr>
<td>CTRL-D</td>
<td>Näytä/piilota tilastopaneeli.</td>
</tr>
<tr>
<td>CTRL-E</td>
<td>Näytä/piilota Eval-paneeli.</td>
</tr>
<tr>
<td>?</td>
<td>Näytä/piilota ohje.</td>
</tr>
</tbody>
</table>
<h3>Näkymävälilehdet</h3>
<table>
<thead>
<tr>
<th>Oikotie</th>
<th>Toiminto</th>
</tr>
</thead>
<tbody>
<tr>
<td>CTRL-T</td>
<td>Luo uusi näkymä (nykyisen näkymän kopio).</td>
</tr>
<tr>
<td>CTRL-W</td>
<td>Sulje nykyinen näkymä.</td>
</tr>
<tr>
<td>CTRL-PageUp, SHIFT-J</td>
<td>Edellinen näkymä.</td>
</tr>
<tr>
<td>CTRL-PageDown, SHIFT-K</td>
<td>Seuraava näkymä.</td>
</tr>
<tr>
<td>CTRL-1 … CTRL-9</td>
<td>Siirry suoraan n:nteen näkymään.</td>
</tr>
<tr>
<td>Välilehden kaksoisnapsautus</td>
<td>Nimeä näkymä uudelleen.</td>
</tr>
</tbody>
</table>
<h3>Komentorivi</h3>
<table>
<thead>
<tr>
<th>Oikotie</th>
<th>Toiminto</th>
</tr>
</thead>
<tbody>
<tr>
<td>YLÖS</td>
<td>Selaa komentohistoriaa ylöspäin.</td>
</tr>
<tr>
<td>ALAS</td>
<td>Selaa komentohistoriaa alaspäin.</td>
</tr>
</tbody>
</table>
<h3>Hakuhistoria</h3>
<table>
<thead>
<tr>
<th>Oikotie</th>
<th>Toiminto</th>
</tr>
</thead>
<tbody>
<tr>
<td>Napsautus</td>
<td>Valitse/poista valinta haulta (näytä asema).</td>
</tr>
<tr>
<td>Kaksoisnapsautus</td>
<td>Suorita haku.</td>
</tr>
</tbody>
</table>
<h3>Suodatinkirjasto</h3>
<table>
<thead>
<tr>
<th>Oikotie</th>
<th>Toiminto</th>
</tr>
</thead>
<tbody>
<tr>
<td>Napsautus</td>
<td>Valitse/poista valinta suodattimelta (näytä asema).</td>
</tr>
<tr>
<td>Kaksoisnapsautus</td>
<td>Suorita suodattimen haku.</td>
</tr>
</tbody>
</table>
<h3>Analyysipaneeli</h3>
<table>
<thead>
<tr>
<th>Oikotie</th>
<th>Toiminto</th>
</tr>
</thead>
<tbody>
<tr>
<td>Napsautus</td>
<td>Valitse/poista valinta siirrolta (näytä/piilota nuolet).</td>
</tr>
<tr>
<td>YLÖS, k</td>
<td>Valitse edellinen siirto (kun siirto on valittu).</td>
</tr>
<tr>
<td>ALAS, j</td>
<td>Valitse seuraava siirto (kun siirto on valittu).</td>
</tr>
<tr>
<td>d</td>
<td>Vaihda nappuloiden ja tuplauskuution analyysin välillä (vain ottelunavigoinnissa).</td>
</tr>
<tr>
<td>Esc</td>
<td>Poista siirron valinta. Jos mitään siirtoa ei ole valittu, sulje paneeli.</td>
</tr>
</tbody>
</table>
<h3>Eval-paneeli</h3>
<table>
<thead>
<tr>
<th>Oikotie</th>
<th>Toiminto</th>
</tr>
</thead>
<tbody>
<tr>
<td>Napsautus</td>
<td>Valitse/poista valinta siirrolta (näytä/piilota nuolet).</td>
</tr>
<tr>
<td>YLÖS, k</td>
<td>Valitse edellinen siirto (kun siirto on valittu).</td>
</tr>
<tr>
<td>ALAS, j</td>
<td>Valitse seuraava siirto (kun siirto on valittu).</td>
</tr>
<tr>
<td>Esc</td>
<td>Poista siirron valinta.</td>
</tr>
</tbody>
</table>
<h3>Ottelupaneeli</h3>
<table>
<thead>
<tr>
<th>Oikotie</th>
<th>Toiminto</th>
</tr>
</thead>
<tbody>
<tr>
<td>Napsautus</td>
<td>Valitse ottelu.</td>
</tr>
<tr>
<td>Kaksoisnapsautus</td>
<td>Navigoi ottelussa.</td>
</tr>
<tr>
<td>YLÖS, k</td>
<td>Valitse edellinen ottelu.</td>
</tr>
<tr>
<td>ALAS, j</td>
<td>Valitse seuraava ottelu.</td>
</tr>
<tr>
<td>ENTER</td>
<td>Lataa valittu ottelu.</td>
</tr>
<tr>
<td>Del</td>
<td>Poista valittu ottelu.</td>
</tr>
<tr>
<td>Esc</td>
<td>Poista valinta / sulje paneeli.</td>
</tr>
</tbody>
</table>
<h3>Anki-paneeli (välitoistoharjoittelu)</h3>
<table>
<thead>
<tr>
<th>Oikotie</th>
<th>Toiminto</th>
</tr>
</thead>
<tbody>
<tr>
<td>VÄLILYÖNTI, Napsautus</td>
<td>Näytä vastaus (aseman tallennettu analyysi).</td>
</tr>
<tr>
<td>1</td>
<td>Arvioi: Uudelleen (epäonnistui, kertaa pian).</td>
</tr>
<tr>
<td>2</td>
<td>Arvioi: Vaikea.</td>
</tr>
<tr>
<td>3</td>
<td>Arvioi: Hyvä.</td>
</tr>
<tr>
<td>4</td>
<td>Arvioi: Helppo.</td>
</tr>
<tr>
<td>p</td>
<td>Näytä/piilota pip-laskuri (sama kuin yleinen pikanäppäin, käytettävissä kertauksen aikana).</td>
</tr>
<tr>
<td>Esc</td>
<td>Lopeta kertaus ja palaa pakkaluetteloon (voidaan jatkaa myöhemmin).</td>
</tr>
</tbody>
</table>
<h3>Turnauspaneeli</h3>
<table>
<thead>
<tr>
<th>Oikotie</th>
<th>Toiminto</th>
</tr>
</thead>
<tbody>
<tr>
<td>Napsautus, Kaksoisnapsautus</td>
<td>Valitse turnaus (näytä sen tiedot).</td>
</tr>
<tr>
<td>YLÖS, k</td>
<td>Valitse edellinen turnaus.</td>
</tr>
<tr>
<td>ALAS, j</td>
<td>Valitse seuraava turnaus.</td>
</tr>
<tr>
<td>Kaksoisnapsautus (turnauksen ottelun kohdalla)</td>
<td>Navigoi ottelussa.</td>
</tr>
<tr>
<td>Esc</td>
<td>Peruuta käynnissä oleva muokkaus, muuten tyhjennä ottelun lisäyshaku, muuten poista turnauksen valinta, muuten sulje paneeli (askel kerrallaan).</td>
</tr>
</tbody>
</table>
<h3>Kokoelmapaneeli</h3>
<table>
<thead>
<tr>
<th>Oikotie</th>
<th>Toiminto</th>
</tr>
</thead>
<tbody>
<tr>
<td>Napsautus</td>
<td>Lisää nykyinen asema osoittimen alla olevaan kokoelmaan tai poista se siitä.</td>
</tr>
<tr>
<td>Kaksoisnapsautus</td>
<td>Avaa kokoelma.</td>
</tr>
<tr>
<td>Del</td>
<td>Poista nykyinen asema (tai valitut asemat) avoimesta kokoelmasta.</td>
</tr>
<tr>
<td>Esc</td>
<td>Palaa kokoelmalistaan, muuten poista kokoelman valinta, muuten sulje paneeli (askel kerrallaan).</td>
</tr>
</tbody>
</table>
<h3>Ohjepaneeli</h3>
<table>
<thead>
<tr>
<th>Oikotie</th>
<th>Toiminto</th>
</tr>
</thead>
<tbody>
<tr>
<td>VASEN, h</td>
<td>Edellinen välilehti.</td>
</tr>
<tr>
<td>OIKEA, l</td>
<td>Seuraava välilehti.</td>
</tr>
<tr>
<td>YLÖS, k</td>
<td>Vieritä ylöspäin.</td>
</tr>
<tr>
<td>ALAS, j</td>
<td>Vieritä alaspäin.</td>
</tr>
<tr>
<td>VÄLILYÖNTI</td>
<td>Seuraava sivu.</td>
</tr>
<tr>
<td>PageUp</td>
<td>Sisällön alkuun.</td>
</tr>
<tr>
<td>PageDown</td>
<td>Sisällön loppuun.</td>
</tr>
<tr>
<td>?, CTRL-F, Esc</td>
<td>Sulje ohje.</td>
</tr>
</tbody>
</table>
`,
    commands: `
<p>Komentorivi, joka sijaitsee tilarivillä, avataan painamalla <em>VÄLILYÖNTI</em>-näppäintä. Komentoa kirjoitettaessa ehdotusluettelo ilmestyy automaattisesti: <em>TAB</em>-näppäin (tai <em>VAIHTO-TAB</em>) selaa ehdotuksia ja täydentää komennon, kun taas <em>ESC</em> sulkee luettelon (toinen <em>ESC</em> sulkee komentorivin). <em>YLÖS</em>- ja <em>ALAS</em>-näppäimet on edelleen varattu komentohistorialle.</p>
<h3>Yleiset toiminnot</h3>
<table>
<thead>
<tr>
<th>Komento</th>
<th>Toiminto</th>
</tr>
</thead>
<tbody>
<tr>
<td>new, ne, n</td>
<td>Luo uuden tietokannan.</td>
</tr>
<tr>
<td>open, op, o</td>
<td>Avaa olemassa olevan tietokannan.</td>
</tr>
<tr>
<td>import_db, idb</td>
<td>Tuo ja yhdistää toisen tietokannan.</td>
</tr>
<tr>
<td>export_db, edb</td>
<td>Vie nykyisen valinnan uuteen tietokantaan.</td>
</tr>
<tr>
<td>quit, q</td>
<td>Sulkee blunderDB:n.</td>
</tr>
<tr>
<td>help, he, h</td>
<td>Avaa blunderDB:n ohjeen.</td>
</tr>
<tr>
<td>tutorial, tour</td>
<td>Avaa käyttöliittymän opastettujen kierrosten luettelon.</td>
</tr>
<tr>
<td>demo</td>
<td>Lataa esimerkkitietokannan (otteluita, turnaus, kokoelmia, kommentteja, Anki-pakka, analyysejä) työkaluun tutustumista varten.</td>
</tr>
<tr>
<td>meta</td>
<td>Näyttää tietokannan metatiedot.</td>
</tr>
<tr>
<td>epc</td>
<td>Avaa Eval-paneelin (Effective Pip Count, voittotodennäköisyys ja kuutiopäätös bearoffissa). <code>epc</code> on tämän paneelin vanha nimi, joka on säilytetty.</td>
</tr>
<tr>
<td>met</td>
<td>Avaa Kazaross-XG2-ottelutaulukon (match equity table).</td>
</tr>
<tr>
<td>cm</td>
<td>Avaa kuutiomatriisin: nykyisen aseman tuomion 5-, 7- tai 9-pisteen ottelun jokaisessa pistetilanteessa.</td>
</tr>
<tr>
<td>tags</td>
<td>Avaa tunnistesanaston: tässä tietokannassa käytetyt tunnisteet asemamäärineen, napsautettavina haun käynnistämiseksi.</td>
</tr>
<tr>
<td>tp2</td>
<td>Avaa take-pisteiden taulukon kuution arvolla 2.</td>
</tr>
<tr>
<td>tp2_live</td>
<td>Avaa take-pisteiden taulukon kuution arvolla 2 pitkille kilpajuoksuille.</td>
</tr>
<tr>
<td>tp2_last</td>
<td>Avaa take-pisteiden taulukon kuution arvolla 2 viimeiselle heitolle.</td>
</tr>
<tr>
<td>tp4</td>
<td>Avaa take-pisteiden taulukon kuution arvolla 4.</td>
</tr>
<tr>
<td>tp4_live</td>
<td>Avaa take-pisteiden taulukon kuution arvolla 4 pitkille kilpajuoksuille.</td>
</tr>
<tr>
<td>tp4_last</td>
<td>Avaa take-pisteiden taulukon kuution arvolla 4 viimeiselle heitolle.</td>
</tr>
<tr>
<td>gv1</td>
<td>Avaa gammon-arvojen taulukon kuution arvolla 1.</td>
</tr>
<tr>
<td>gv2</td>
<td>Avaa gammon-arvojen taulukon kuution arvolla 2.</td>
</tr>
<tr>
<td>gv4</td>
<td>Avaa gammon-arvojen taulukon kuution arvolla 4.</td>
</tr>
</tbody>
</table>
<h3>Asemat ja navigointi</h3>
<table>
<thead>
<tr>
<th>Komento</th>
<th>Toiminto</th>
</tr>
</thead>
<tbody>
<tr>
<td>import, i</td>
<td>Tuo yhden tai useamman aseman/ottelun tiedostosta (xg, xgp, sgf, mat, txt, bgf). Argumentin kanssa — <code>import XGID=…</code> — lukee tunnisteen tiedostovalitsimen avaamisen sijaan, kun se tulee viestistä, foorumilta tai skriptistä.</td>
</tr>
<tr>
<td>delete, del, d</td>
<td>Poistaa nykyisen aseman (vahvistus pyydetään); poisto kulkee roskakorin kautta ja on peruttavissa kolmenkymmenen päivän ajan.</td>
</tr>
<tr>
<td>trash</td>
<td>Avaa roskakorin: mitä on poistettu ja millä se palautetaan.</td>
</tr>
<tr>
<td>[number]</td>
<td>Siirry annetun indeksin asemaan.</td>
</tr>
<tr>
<td>list, l</td>
<td>Näytä nykyisen aseman analyysi.</td>
</tr>
<tr>
<td>comment, co</td>
<td>Näytä/kirjoita kommentteja.</td>
</tr>
<tr>
<td>history, hi</td>
<td>Avaa hakupaneeli (hakuhistoria löytyy sen <em>Historique</em>-välilehdeltä).</td>
</tr>
<tr>
<td>stats, st</td>
<td>Näytä/piilota tilastopaneeli.</td>
</tr>
<tr>
<td>match, ma</td>
<td>Näytä/piilota otteluiden paneeli.</td>
</tr>
<tr>
<td>collection, coll</td>
<td>Näytä/piilota kokoelmien paneeli.</td>
</tr>
<tr>
<td>#tag1 tag2 ...</td>
<td>Merkitse nykyinen asema tunnisteilla.</td>
</tr>
<tr>
<td>e</td>
<td>Lataa kaikki tietokannan asemat.</td>
</tr>
<tr>
<td>blunders, bl [n]</td>
<td>Lataa pahimmat virheet (equity/MWC) analyysinäkymään nykyisen tilastosuodattimen mukaisesti. Valinnainen luku valitsee, kuinka monta ladataan (<code>bl 50</code>); oletuksena 10.Lataa pahimmat virheet (equity/MWC) analyysinäkymään nykyisen tilastosuodattimen mukaisesti.</td>
</tr>
<tr>
<td>m</td>
<td>Navigoi viimeksi käytyyn otteluun.</td>
</tr>
</tbody>
</table>
<h3>Muokkaus ja haku</h3>
<table>
<thead>
<tr>
<th>Komento</th>
<th>Toiminto</th>
</tr>
</thead>
<tbody>
<tr>
<td>write, wr, w</td>
<td>Tallentaa nykyisen aseman.</td>
</tr>
<tr>
<td>write!, wr!, w!</td>
<td>Päivittää nykyisen aseman.</td>
</tr>
<tr>
<td>s</td>
<td>Etsi asemia suodattimilla.</td>
</tr>
<tr>
<td>ss</td>
<td>Etsi tällä hetkellä suodatettujen asemien joukosta.</td>
</tr>
</tbody>
</table>
<h3>Hakusuodattimet</h3>
<p>Tämä taulukko on hakukieliopin viite: komentorivi, suodatinkirjasto ja <code>blunderdb search</code> -komennon valitsin <code>--query</code> lukevat kaikki samoja tunnuksia. Sarake <em>CLI-vastine</em> antaa, silloin kun sellainen on olemassa, saman asian tekevän <code>search</code>-valitsimen (ks. Komentoriviliittymä (CLI)); viiva merkitsee suodatinta, jonka vain kielioppi osaa ilmaista.</p>
<p>Viisi tunnusta ei kanna omaa arvoaan: ne lukevat sen hakulaudalta. <code>cube</code> ja <code>score</code> ottavat sille asetetun kuution ja tuloksen, <code>d</code> päätöksen tyypin, <code>D</code> ja <code>D1</code> nopat, <code>x</code> <em>Paitsi</em>-välilehdellä piirretyn rakenteen. Heittoa ei siis koskaan kirjoiteta tunnukseen: <code>D65</code> ei ole olemassa, ja vain poissulkeva muoto kantaa numeronsa (<code>xD65</code>). Komentorivillä, jossa lautaa ei ole, nämä tunnukset vertautuvat tyhjään lautaan; siellä on käytettävä kolmannen sarakkeen valitsimia.</p>
<p>Virheet ja equityt lasketaan <strong>equityn tuhannesosina</strong> — alla olevan taulukon <em>millipisteinä</em>: <code>E&gt;100</code> poimii siirrot, jotka ovat maksaneet vähintään kymmenesosan pisteestä, sillä yksi piste on 1000 tuhannesosaa.</p>
<p>Kaksi täydellistä hakua:</p>
<ul>
<li><code>s p&gt;30 w40,60 xco</code> — yli 30 pipiä jäljessä, voittomahdollisuus 40–60 %, ei yhtään kommenttia.</li>
<li><code>s ph:race E&gt;50 co:xg</code> — juoksuvaiheessa, siirto joka on maksanut vähintään 50 tuhannesosaa, ja eXtreme Gammonista tullut kommentti.</li>
</ul>
<table>
<thead>
<tr>
<th>Kysely</th>
<th>Toiminto</th>
<th>CLI-vastine</th>
</tr>
</thead>
<tbody>
<tr>
<td>cube, cub, cu, c</td>
<td>Asema vastaa kuution kokoonpanoa.</td>
<td><code>--cube</code></td>
</tr>
<tr>
<td>score, sco, sc, s</td>
<td>Asema vastaa pistetilannetta.</td>
<td><code>--score1</code> <code>--score2</code></td>
</tr>
<tr>
<td>d</td>
<td>Asema vastaa päätöstyyppiä (nappula- tai kuutiopäätös).</td>
<td><code>--decision</code></td>
</tr>
<tr>
<td>D</td>
<td>Asema vastaa nopanheittoa (molemmat nopat, järjestyksestä riippumatta).</td>
<td><code>--dice 6,5</code></td>
</tr>
<tr>
<td>D1</td>
<td>Asema vastaa nopanheittoa vain ensimmäisen nopan osalta (ensimmäisen nopan arvo esiintyy jommassakummassa aseman nopassa).</td>
<td><code>--dice 6</code></td>
</tr>
<tr>
<td>xD65</td>
<td>Asemaa <strong>ei</strong> ole pelattu heitolla 6-5 (järjestyksestä riippumatta). Arvo ilmoitetaan tunnuksessa; toistettavissa useiden heittojen poissulkemiseksi (<code>xD65 xD54</code>).</td>
<td>—</td>
</tr>
<tr>
<td>nc</td>
<td>Asema on ilman kontaktia.</td>
<td>—</td>
</tr>
<tr>
<td>ph:race</td>
<td>Asema on tietyssä pelin vaiheessa: <code>opening</code> (avaus), <code>middlegame</code> (keskipeli), <code>race</code> (kilpajuoksu) tai <code>bearoff</code> (nappuloiden poisto). Toistettavissa (<code>ph:race ph:bearoff</code>). Merkintä johdetaan laudasta eikä sitä voi koskaan muokata; <code>blunderdb repair</code> laskee sen uudelleen.</td>
<td><code>--phase</code></td>
</tr>
<tr>
<td>#prime</td>
<td>Asema kantaa tätä <strong>tunnistetta</strong> jossakin kommentissaan. Tunniste on proosaan kirjoitettu <code>#sana</code>; mikään ei ilmoita sitä. Vertailu on rajattu, joten <code>#prime</code> ei löydä sanaa <code>#priming</code> — juuri siinä on ero tekstisuodattimeen, joka etsii osamerkkijonoa. Toistettavissa, ja tunnisteet <strong>kasautuvat</strong> (<code>#prime #backgame</code> pyytää molempia): asema kantaa useita tunnisteita, joten kahden nimeäminen tarkoittaa ”molempia”.</td>
<td>—</td>
</tr>
<tr>
<td>M</td>
<td>Asema tai sen peilikuva vastaa suodattimia.</td>
<td>—</td>
</tr>
<tr>
<td>i</td>
<td>Asema on tuotu erikseen, eikä se tullut ottelun tuonnin mukana.</td>
<td><code>--individual</code></td>
</tr>
<tr>
<td>fl</td>
<td>Asema on merkitty lähdeohjelmassa eXtreme Gammon -ottelun tuonnin yhteydessä.</td>
<td><code>--flagged</code></td>
</tr>
<tr>
<td>x</td>
<td>Asema ei sisällä yhtäkään poissulkurakenteen nappulaa (hakupaneelin "Except"-välilehti).</td>
<td>—</td>
</tr>
<tr>
<td>p&gt;x</td>
<td>Pelaaja on kilpajuoksussa vähintään x pippiä jäljessä.</td>
<td><code>--pip-min</code></td>
</tr>
<tr>
<td>p&lt;x</td>
<td>Pelaaja on kilpajuoksussa enintään x pippiä jäljessä.</td>
<td><code>--pip-max</code></td>
</tr>
<tr>
<td>px,y</td>
<td>Pelaaja on kilpajuoksussa x–y pippiä jäljessä.</td>
<td><code>--pip-min</code> <code>--pip-max</code></td>
</tr>
<tr>
<td>P&gt;x</td>
<td>Pelaajalla on kilpajuoksu vähintään x pippiä.</td>
<td>—</td>
</tr>
<tr>
<td>P&lt;x</td>
<td>Pelaajalla on kilpajuoksu enintään x pippiä.</td>
<td>—</td>
</tr>
<tr>
<td>Px,y</td>
<td>Pelaajalla on kilpajuoksu x–y pippiä.</td>
<td>—</td>
</tr>
<tr>
<td>e&gt;x</td>
<td>Aseman ekviteetti (millipisteinä) on suurempi kuin x.</td>
<td>—</td>
</tr>
<tr>
<td>e&lt;x</td>
<td>Aseman ekviteetti (millipisteinä) on pienempi kuin x.</td>
<td>—</td>
</tr>
<tr>
<td>ex,y</td>
<td>Aseman ekviteetti (millipisteinä) on välillä x–y.</td>
<td>—</td>
</tr>
<tr>
<td>E&gt;x</td>
<td>Pelaajan 1 tekemän siirron virhe (millipisteinä) on suurempi kuin x.</td>
<td><code>--move-error-min</code></td>
</tr>
<tr>
<td>E&lt;x</td>
<td>Pelaajan 1 tekemän siirron virhe (millipisteinä) on pienempi kuin x.</td>
<td><code>--move-error-max</code></td>
</tr>
<tr>
<td>Ex,y</td>
<td>Pelaajan 1 tekemän siirron virhe (millipisteinä) on välillä x–y.</td>
<td><code>--move-error-min</code> <code>--move-error-max</code></td>
</tr>
<tr>
<td>w&gt;x</td>
<td>Pelaajan voittomahdollisuudet ovat suuremmat kuin x %.</td>
<td><code>--winrate-min</code></td>
</tr>
<tr>
<td>w&lt;x</td>
<td>Pelaajan voittomahdollisuudet ovat pienemmät kuin x %.</td>
<td><code>--winrate-max</code></td>
</tr>
<tr>
<td>wx,y</td>
<td>Pelaajan voittomahdollisuudet ovat välillä x % – y %.</td>
<td><code>--winrate-min</code> <code>--winrate-max</code></td>
</tr>
<tr>
<td>g&gt;x</td>
<td>Pelaajan gammon-mahdollisuudet ovat suuremmat kuin x %.</td>
<td>—</td>
</tr>
<tr>
<td>g&lt;x</td>
<td>Pelaajan gammon-mahdollisuudet ovat pienemmät kuin x %.</td>
<td>—</td>
</tr>
<tr>
<td>gx,y</td>
<td>Pelaajan gammon-mahdollisuudet ovat välillä x % – y %.</td>
<td>—</td>
</tr>
<tr>
<td>b&gt;x</td>
<td>Pelaajan backgammon-mahdollisuudet ovat suuremmat kuin x %.</td>
<td>—</td>
</tr>
<tr>
<td>b&lt;x</td>
<td>Pelaajan backgammon-mahdollisuudet ovat pienemmät kuin x %.</td>
<td>—</td>
</tr>
<tr>
<td>bx,y</td>
<td>Pelaajan backgammon-mahdollisuudet ovat välillä x % – y %.</td>
<td>—</td>
</tr>
<tr>
<td>W&gt;x</td>
<td>Vastustajan voittomahdollisuudet ovat suuremmat kuin x %.</td>
<td>—</td>
</tr>
<tr>
<td>W&lt;x</td>
<td>Vastustajan voittomahdollisuudet ovat pienemmät kuin x %.</td>
<td>—</td>
</tr>
<tr>
<td>Wx,y</td>
<td>Vastustajan voittomahdollisuudet ovat välillä x % – y %.</td>
<td>—</td>
</tr>
<tr>
<td>G&gt;x</td>
<td>Vastustajan gammon-mahdollisuudet ovat suuremmat kuin x %.</td>
<td>—</td>
</tr>
<tr>
<td>G&lt;x</td>
<td>Vastustajan gammon-mahdollisuudet ovat pienemmät kuin x %.</td>
<td>—</td>
</tr>
<tr>
<td>Gx,y</td>
<td>Vastustajan gammon-mahdollisuudet ovat välillä x % – y %.</td>
<td>—</td>
</tr>
<tr>
<td>B&gt;x</td>
<td>Vastustajan backgammon-mahdollisuudet ovat suuremmat kuin x %.</td>
<td>—</td>
</tr>
<tr>
<td>B&lt;x</td>
<td>Vastustajan backgammon-mahdollisuudet ovat pienemmät kuin x %.</td>
<td>—</td>
</tr>
<tr>
<td>Bx,y</td>
<td>Vastustajan backgammon-mahdollisuudet ovat välillä x % – y %.</td>
<td>—</td>
</tr>
<tr>
<td>o&gt;x</td>
<td>Pelaajalla on vähintään x ulos kannettua nappulaa.</td>
<td><code>--off1-min</code></td>
</tr>
<tr>
<td>o&lt;x</td>
<td>Pelaajalla on enintään x ulos kannettua nappulaa.</td>
<td>—</td>
</tr>
<tr>
<td>ox,y</td>
<td>Pelaajalla on x–y ulos kannettua nappulaa.</td>
<td>—</td>
</tr>
<tr>
<td>O&gt;x</td>
<td>Vastustajalla on vähintään x ulos kannettua nappulaa.</td>
<td><code>--off2-min</code></td>
</tr>
<tr>
<td>O&lt;x</td>
<td>Vastustajalla on enintään x ulos kannettua nappulaa.</td>
<td>—</td>
</tr>
<tr>
<td>Ox,y</td>
<td>Vastustajalla on x–y ulos kannettua nappulaa.</td>
<td>—</td>
</tr>
<tr>
<td>k&gt;x</td>
<td>Pelaajalla on vähintään x takanappulaa.</td>
<td>—</td>
</tr>
<tr>
<td>k&lt;x</td>
<td>Pelaajalla on enintään x takanappulaa.</td>
<td>—</td>
</tr>
<tr>
<td>kx,y</td>
<td>Pelaajalla on x–y takanappulaa.</td>
<td>—</td>
</tr>
<tr>
<td>K&gt;x</td>
<td>Vastustajalla on vähintään x takanappulaa.</td>
<td>—</td>
</tr>
<tr>
<td>K&lt;x</td>
<td>Vastustajalla on enintään x takanappulaa.</td>
<td>—</td>
</tr>
<tr>
<td>Kx,y</td>
<td>Vastustajalla on x–y takanappulaa.</td>
<td>—</td>
</tr>
<tr>
<td>z&gt;x</td>
<td>Pelaajalla on vähintään x nappulaa vyöhykkeellä.</td>
<td>—</td>
</tr>
<tr>
<td>z&lt;x</td>
<td>Pelaajalla on enintään x nappulaa vyöhykkeellä.</td>
<td>—</td>
</tr>
<tr>
<td>zx,y</td>
<td>Pelaajalla on x–y nappulaa vyöhykkeellä.</td>
<td>—</td>
</tr>
<tr>
<td>Z&gt;x</td>
<td>Vastustajalla on vähintään x nappulaa vyöhykkeellä.</td>
<td>—</td>
</tr>
<tr>
<td>Z&lt;x</td>
<td>Vastustajalla on enintään x nappulaa vyöhykkeellä.</td>
<td>—</td>
</tr>
<tr>
<td>Zx,y</td>
<td>Vastustajalla on x–y nappulaa vyöhykkeellä.</td>
<td>—</td>
</tr>
<tr>
<td>bo&gt;x</td>
<td>Pelaajalla on vähintään x blotia ulkokentällä.</td>
<td>—</td>
</tr>
<tr>
<td>bo&lt;x</td>
<td>Pelaajalla on enintään x blotia ulkokentällä.</td>
<td>—</td>
</tr>
<tr>
<td>box,y</td>
<td>Pelaajalla on x–y blotia ulkokentällä.</td>
<td>—</td>
</tr>
<tr>
<td>BO&gt;x</td>
<td>Vastustajalla on vähintään x blotia ulkokentällä.</td>
<td>—</td>
</tr>
<tr>
<td>BO&lt;x</td>
<td>Vastustajalla on enintään x blotia ulkokentällä.</td>
<td>—</td>
</tr>
<tr>
<td>BOx,y</td>
<td>Vastustajalla on x–y blotia ulkokentällä.</td>
<td>—</td>
</tr>
<tr>
<td>bj&gt;x</td>
<td>Pelaajalla on vähintään x blotia kotialueella.</td>
<td>—</td>
</tr>
<tr>
<td>bj&lt;x</td>
<td>Pelaajalla on enintään x blotia kotialueella.</td>
<td>—</td>
</tr>
<tr>
<td>bjx,y</td>
<td>Pelaajalla on x–y blotia kotialueella.</td>
<td>—</td>
</tr>
<tr>
<td>BJ&gt;x</td>
<td>Vastustajalla on vähintään x blotia kotialueella.</td>
<td>—</td>
</tr>
<tr>
<td>BJ&lt;x</td>
<td>Vastustajalla on enintään x blotia kotialueella.</td>
<td>—</td>
</tr>
<tr>
<td>BJx,y</td>
<td>Vastustajalla on x–y blotia kotialueella.</td>
<td>—</td>
</tr>
<tr>
<td><code>t'sana1;sana2;...'</code></td>
<td>Aseman kommentit sisältävät vähintään yhden sanoista.</td>
<td>—</td>
</tr>
<tr>
<td>co</td>
<td>Asemassa on kommentti, sisällöstä riippumatta.</td>
<td><code>--has-comment</code></td>
</tr>
<tr>
<td>xco</td>
<td>Asemassa ei ole kommenttia.</td>
<td><code>--no-comment</code></td>
</tr>
<tr>
<td>co:user</td>
<td>Asemaan liittyy tietystä lähteestä peräisin oleva kommentti: <code>user</code> (sinun kirjoittamasi), <code>xg</code>, <code>gnubg</code>, <code>bgf</code> (ottelun tuonnin mukana tullut) tai <code>unknown</code>. Toistettavissa (<code>co:xg co:gnubg</code>).</td>
<td><code>--comment-origin</code></td>
</tr>
<tr>
<td><code>m'kuvio1,kuvio2,...'</code></td>
<td>Parhaat nappulasiirrot, jotka sisältävät vähintään yhden kuvioista.</td>
<td>—</td>
</tr>
<tr>
<td><code>m'ND,DT,DP,...'</code></td>
<td>Parhaat kuutiopäätökset No Double/Take, Double Take, Double Pass.</td>
<td>—</td>
</tr>
<tr>
<td>T&gt;x</td>
<td>Aseman lisäyspäivä x:n jälkeen (VVVV/KK/PP).</td>
<td>—</td>
</tr>
<tr>
<td>T&lt;x</td>
<td>Aseman lisäyspäivä ennen x:ää (VVVV/KK/PP).</td>
<td>—</td>
</tr>
<tr>
<td>Tx,y</td>
<td>Aseman lisäyspäivä välillä x–y (VVVV/KK/PP).</td>
<td>—</td>
</tr>
<tr>
<td>max</td>
<td>Etsi ottelusta, jonka tunnus on x (esim. ma3).</td>
<td><code>--match-ids</code></td>
</tr>
<tr>
<td>max,y</td>
<td>Etsi otteluista, joiden tunnukset ovat x–y (esim. ma2,5).</td>
<td><code>--match-ids</code></td>
</tr>
<tr>
<td>tnx</td>
<td>Etsi turnauksesta, jonka tunnus on x (esim. tn1).</td>
<td><code>--tournament-ids</code></td>
</tr>
<tr>
<td>tnx,y</td>
<td>Etsi turnauksista, joiden tunnukset ovat x–y (esim. tn1,3).</td>
<td><code>--tournament-ids</code></td>
</tr>
<tr>
<td>idx</td>
<td>Hae asemaa, jonka tunnus on x (esim. id12).</td>
<td><code>--position-ids</code></td>
</tr>
<tr>
<td>idx,y</td>
<td>Hae asemia, joiden tunnukset ovat välillä x–y (esim. id5,10).</td>
<td><code>--position-ids</code></td>
</tr>
<tr>
<td><code>pl'nimi'</code></td>
<td>Hae asemia ottelusta, jossa nimetty pelaaja oli mukana kummalla tahansa puolella (esim. <code>pl'Alice'</code>). Kirjainkoolla ei ole väliä.</td>
<td>—</td>
</tr>
</tbody>
</table>
<h3>Sekalaisia komentoja</h3>
<table>
<thead>
<tr>
<th>Komento</th>
<th>Toiminto</th>
</tr>
</thead>
<tbody>
<tr>
<td>clear, cl</td>
<td>Tyhjentää komentohistorian.</td>
</tr>
</tbody>
</table>
`,
    about: `
<h3>Versio</h3>
<p>Sovelluksen versio: {appVersion}</p>
<p>Tietokannan versio: {dbVersion}</p>
<p>
    <a href="https://kevung.github.io/blunderDB/fi/" target="_blank" rel="noopener noreferrer">Verkkodokumentaatio</a> ·
    <a href="https://kevung.github.io/blunderDB/fi/historique.html" target="_blank" rel="noopener noreferrer">Versiohistoria</a>
</p>

<h3>Tekijä</h3>
<p><strong>Kévin Unger &lt;blunderdb@proton.me&gt;</strong></p>
<p>Minut löytää myös Heroesista nimimerkillä <strong>postmanpat</strong>.</p>
<p>
    Kehitin blunderDB:n alun perin omaan käyttööni havaitakseni kaavoja virheissäni. Mutta on erittäin mukavaa saada palautetta, etenkin kun suunnitteluun, koodaamiseen ja virheenkorjaukseen on
    käytetty paljon tunteja... Joten kirjoita minulle vapaasti jakaaksesi palautteesi.
</p>
<p>Tässä useita tapoja ottaa yhteyttä:</p>
<ul>
    <li>Liity blunderDB:n Discord-palvelimelle: <a href="https://discord.gg/DA5PpzM9En" target="_blank" rel="noopener noreferrer">discord.gg/DA5PpzM9En</a>,</li>
    <li>Keskustele kanssani, jos tapaamme turnauksessa,</li>
    <li>Lähetä minulle sähköpostia,</li>
</ul>
<h3>Lisenssi</h3>
<p>
    blunderDB on lisensoitu MIT-lisenssillä. Tämä tarkoittaa, että voit vapaasti käyttää, kopioida, muokata, yhdistää, julkaista, jakaa, alilisensoida ja/tai myydä ohjelmiston kopioita edellyttäen,
    että alkuperäinen tekijänoikeusilmoitus ja tämä lupailmoitus sisällytetään kaikkiin kopioihin tai ohjelmiston olennaisiin osiin.
</p>
<h3>Kiitokset</h3>
<p>Omistan tämän pienen ohjelmiston kumppanilleni <strong>Anne-Clairelle</strong> ja rakkaalle tyttärellemme <strong>Perrinelle</strong>. Haluan kiittää erityisesti muutamia ystäviä:</p>
<ul>
    <li>
        <strong>Tristan Remille</strong>, joka esitteli minulle backgammonin ilolla ja ystävällisyydellä; joka näytti Tien tämän upean pelin ymmärtämiseen; joka jatkaa tukemistani huolimatta huonoista
        yrityksistäni pelata paremmin.
    </li>
    <li><strong>Nicolas Harmand</strong>, iloinen seuralainen yli vuosikymmenen ajan suurissa seikkailuissa ja loistava pelikumppani siitä lähtien, kun hän sai backgammon-kärpäsen.</li>
</ul>
<h3>Kiitokset kolmansille osapuolille</h3>
<p>blunderDB sisältää muiden ihmisten koodia, dataa ja fontteja. Olennaisin:</p>
<ul>
    <li>
        Neuroverkko <strong>strehl-prob5-512-512-256-128</strong> on <strong>Alexander Strehlin</strong> työtä (<em>alexstrehl/backgammon-ai-engine</em>, MIT). Haku, tuplausmalli ja
        otteluekvivalenssitaulukko sen ympärillä ovat <strong>gammonNet</strong>-projektin omaa kokoonpanoa (<a href="https://github.com/kevung/gammonNet" target="_blank" rel="noopener noreferrer"
            >github.com/kevung/gammonNet</a
        >, MIT).
    </li>
    <li>Kazaross-XG2-otteluekvivalenssitaulukko (MET) on <strong>Neil Kazarossin</strong> työtä.</li>
    <li>Take point- ja gammon-arvotaulukot on otettu <strong>Dirk Schiemannin</strong> kirjasta <em>The Theory of Backgammon</em>.</li>
    <li>
        Yksipuolinen (6 pistettä, 15 nappulaa, EPC:tä varten) ja kaksipuolinen (6 pistettä, 6 nappulaa, kilpajuoksujen kuutiopäätöksiä varten) bearoff-tietokanta on luotu
        <strong>GNU Backgammonilla</strong> (GNUbg). GNUbg on GPL-lisensoitu vapaa ohjelmisto; nämä taulukot ovat sen tuottamaa dataa ja ne mainitaan sellaisina.
    </li>
    <li>Ottelutiedostot luetaan kirjastoilla <em>xgparser</em>, <em>gnubgparser</em> ja <em>bgfparser</em> (MIT).</li>
    <li>Go-puolella: <em>modernc.org/sqlite</em> (BSD-3-Clause), <em>pgx</em>, <em>Wails</em> ja <em>go-fsrs</em> (MIT).</li>
    <li>Käyttöliittymän puolella: <em>Svelte</em>, <em>two.js</em>, <em>Chart.js</em> ja <em>driver.js</em> (MIT).</li>
    <li>Fontit <em>Nunito</em> ja <em>Noto Sans JP</em> (SIL Open Font License 1.1).</li>
</ul>
<p>
    Täydellinen luettelo lisenssiteksteineen on blunderDB:n mukana toimitettava tiedosto <strong>THIRD_PARTY.md</strong> (<a
        href="https://github.com/kevung/blunderDB/blob/main/THIRD_PARTY.md"
        target="_blank"
        rel="noopener noreferrer"
        >github.com/kevung/blunderDB</a
    >).
</p>
`
};
