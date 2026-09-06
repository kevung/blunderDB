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
<h3>Johdanto</h3>
<p>blunderDB on ohjelmisto backgammon-asemien tietokantojen luomiseen. Sen tärkein vahvuus on tarjota yksi paikka koota asemat, joita pelaaja on kohdannut (verkossa, turnauksissa), ja mahdollisuus tutkia näitä asemia uudelleen suodattamalla niitä erilaisilla mielivaltaisesti yhdisteltävillä suodattimilla. blunderDB:tä voi käyttää myös viiteasemien luettelojen luomiseen.</p>
<p>Asemat tallennetaan tietokantaan, jota edustaa <em>.db</em>-tiedosto. Työpöytäsovellus avaa tämän tiedoston suoraan, ei koskaan verkko-osoitetta: palvelintila (Headless-tila (palvelin)) on saman binäärin toinen tila, ja toisesta toiseen siirrytään viemällä tai migroimalla tietokanta, ei osoittamalla sovellusta URL-osoitteeseen.</p>
<h3>Päätoiminnot</h3>
<p>blunderDB:n keskeiset mahdolliset toiminnot ovat:</p>
<ul>
<li>lisätä uusi asema,</li>
<li>muokata olemassa olevaa asemaa,</li>
<li>kopioida laudan kuva leikepöydälle (PNG) näppäimillä <strong>CTRL-X</strong>, tai täydellisen analyysin kanssa näppäimillä <strong>CTRL-X CTRL-X</strong>,</li>
<li>poistaa olemassa oleva asema,</li>
<li>hakea yksi tai useampi asema,</li>
<li>tuoda otteluita eri lähteistä (XG, GNUbg, BGBlitz, Jellyfish), mukaan lukien kommentit XG-tiedostoista,</li>
<li>navigoida tuodun ottelun siirroissa,</li>
<li>järjestää asemat kokoelmiin,</li>
<li>järjestää ottelut turnauksiin.</li>
</ul>
<p>Käyttäjä voi vapaasti merkitä asemat tunnisteilla ja varustaa ne kommenteilla.</p>
<h3>Käyttöliittymän kuvaus</h3>
<p>blunderDB:n käyttöliittymä koostuu ylhäältä alas seuraavista:</p>
<ul>
<li>[ylhäällä] työkalupalkki, joka kokoaa kaikki tietokantaan kohdistuvat keskeiset toiminnot,</li>
<li>[keskellä] pääesitysalue, joka mahdollistaa backgammon-asemien näyttämisen tai muokkaamisen,</li>
<li>[alhaalla] tilarivi, joka esittää erilaisia tietoja tietokannasta tai nykyisestä asemasta ja sisältää komentorivin.</li>
</ul>
<p>Paneeleita voidaan näyttää seuraaviin tarkoituksiin:</p>
<ul>
<li>näyttää nykyiseen asemaan liittyvät analyysitiedot lähteistä eXtreme Gammon (XG), GNUbg tai BGBlitz,</li>
<li>näyttää, lisätä tai muokata kommentteja,</li>
<li>hakea ja suodattaa asemia yhdisteltävillä kriteereillä,</li>
<li>näyttää ja hallita asemakokoelmia (kokoelmapaneeli),</li>
<li>näyttää tuotujen otteluiden luettelo ja navigoida ottelun siirroissa (ottelupaneeli),</li>
<li>näyttää ja hallita turnauksia (turnauspaneeli),</li>
<li>näyttää suoritustilastot (Stats-paneeli),</li>
<li>laskea bearoff-aseman EPC (Effective Pip Count) (Eval-paneeli),</li>
<li>opiskella asemia välitoistolla (Anki-paneeli),</li>
<li>näyttävät tietokannan metatiedot (Metatiedot-paneeli).</li>
</ul>
<p>Modaali-ikkunoita voidaan näyttää seuraaviin tarkoituksiin:</p>
<ul>
<li>näyttää blunderDB:n ohje,</li>
<li>näyttää opastettujen kierrosten luettelon (katso Opastetut kierrokset ja esimerkkitietokanta),</li>
<li>määrittää tietokannan viennin asetukset,</li>
<li>määrittää blunderDB:n asetuksia, erityisesti käyttöliittymän kielen (katso Asetukset).</li>
</ul>
<p>Pääesitysalue tarjoaa käyttäjälle:</p>
<ul>
<li>laudan backgammon-aseman näyttämiseen tai muokkaamiseen,</li>
<li>kuution tason ja omistajan,</li>
<li>kunkin pelaajan pip-luvun,</li>
<li>kunkin pelaajan pistetilanteen,</li>
<li>pelattavat nopat. Jos nopilla ei näy arvoja, noppien sijainti osoittaa, kummalla pelaajalla on vuoro ja että asema on kuutiopäätös. Kun kuutiopäätös on vastaus tuplaukseen (hyväksy/luovuta), tarjottu kuutio näytetään laudan keskellä tarjotulla arvolla.</li>
</ul>
<p>Hiiren oikea napsautus laudalla avaa valikon, joka tarjoaa: näytetyn aseman arvioinnin Eval-paneelissa, sen peilikuvan arvioinnin, laudan kuvan ja sen analyysin kopioinnin leikepöydälle (<em>CTRL-X CTRL-X</em>:n vastine, vaikeampi löytää), <strong>kuvan tallennuksen tiedostoon</strong> SVG- tai PNG-muodossa, uuden näkymän avaamisen tähän asemaan, ja — jos asema tulee jo tietokannasta — sen lisäämisen Anki-pakkaan (välein toistaminen).</p>
<p>Leikepöytä on arkinen ele; tallentaminen on se toinen tarve — kuvitus artikkeliin, foorumiviestiin, oppituntiin. <strong>SVG</strong> tarjotaan siksi, että lauta on sellainen: se on muoto joka kestää suurentamisen, se jonka voi panna asiakirjaan sumentumatta. PNG johdetaan siitä, kuten leikepöydälle kopiointikin: yksi renderöinti, kolme määränpäätä, joten yksikään ei voi ajautua muista erilleen. Tämä valikko ei ilmesty Eval-paneelissa eikä Haku-paneelissa, joissa oikea painike jo asettaa toisen värin nappuloita. Katso Aseman tuominen Eval-paneeliin aseman tuomisesta Eval-paneeliin.</p>
<p>Tilarivi on jäsennelty vasemmalta oikealle seuraavin tiedoin:</p>
<ul>
<li>komentorivi, joka avataan painamalla <em>VÄLILYÖNTI</em>-näppäintä,</li>
<li>käyttäjän suorittamaan toimintoon liittyvä tiedotusviesti,</li>
<li>nykyisen aseman järjestysnumeron, jota seuraa asemien määrä nykyisessä kirjastossa (tai siirto-/pelitiedot ottelua selattaessa),</li>
<li><strong>kirjastolaskurin</strong> — ”412 asemaa · 38 blunderia · 5 ottelua” — jossa jokainen luku <strong>avaa sen, mitä se laskee</strong>: asemat, komentoriville valmisteltu haku <code>E&gt;100</code> tai otteluluettelon. Luku, jota ei voi seurata, on koriste. Blunderin kynnys on tilastojen oma, sata millipistettä: kaksi kynnystä saisi saman sanan tarkoittamaan kahta asiaa.</li>
</ul>
<div class="admonition note">
<p>Käyttäjän haun tuloksena saaduissa asemissa tilarivillä näkyvä asemien määrä vastaa suodatettujen asemien määrää.</p>
</div>
<p><strong>Anki</strong>-välilehti kantaa <strong>merkkiä</strong>, kun kortteja on kerrattavana, kaikki pakat mukaan lukien. Tuo luku on syy avata välilehti; sillä ei ole asiaa sen taakse. Nolla ei näytä mitään: ”0”:aa näyttävä merkki on kohinaa.</p>
<p>Komento <code>log</code> avaa <strong>toimintalokin</strong>: lokitiedoston kaksisataa viimeistä riviä, painikkeen niiden kopioimiseen — juuri sen mitä raportin liittäminen ilmoitukseen vaatii — ja toisen kansion avaamiseen. Lokia ei suodateta eikä muotoilla uudelleen: siistitty loki ei enää kelpaa lainaukseksi.</p>
<p>Hakupaneelin <strong>hakuhistoriassa</strong> tallennetun komennon jokainen tunnus näkyy nimettynä merkkinä — <em>Ei kontaktia</em>, <em>Siirtovirhe</em> — paljaan tunnuksen sijaan. Tarkka komento jää työkaluvihjeeseen, sillä juuri se ajetaan uudelleen; ja tunnus jota blunderDB ei tunnista näkyy <strong>sellaisenaan</strong> eikä lähimmäksi käännettynä.</p>
<h3>Näkymävälilehdet</h3>
<p>Työkalupalkin alla oleva välilehtipalkki mahdollistaa työskentelyn useiden <strong>näkymien</strong> kanssa rinnakkain. Jokainen näkymä on itsenäinen työtila, joka säilyttää oman asemaluettelonsa, nykyisen aseman indeksin, näytetyn aseman, analyysin ja valitun siirron, aktiivisen paneelin, käynnissä olevan kommentin sekä ottelun navigointikontekstin. Näin voi esimerkiksi pitää haun auki yhdessä näkymässä ja selata samalla ottelua toisessa.</p>
<ul>
<li><strong>Näkymän luominen</strong>: napsauta välilehtipalkin <em>+</em>-painiketta tai paina <em>CTRL-T</em>. Uusi näkymä käynnistyy nykyisen näkymän kopiona.</li>
<li><strong>Näkymän sulkeminen</strong>: napsauta välilehden ruksia tai paina <em>CTRL-W</em>. Viimeistä näkymää ei voi sulkea.</li>
<li><strong>Näkymän vaihtaminen</strong>: napsauta välilehteä, paina <em>CTRL-PageUp</em> / <em>CTRL-PageDown</em> (tai <em>SHIFT-J</em> / <em>SHIFT-K</em>) siirtyäksesi edelliseen / seuraavaan näkymään, tai <em>CTRL-1</em> – <em>CTRL-9</em> siirtyäksesi suoraan n:nteen näkymään.</li>
<li><strong>Näkymän uudelleennimeäminen</strong>: kaksoisnapsauta välilehteä, kirjoita uusi nimi ja vahvista painamalla <em>ENTER</em>.</li>
</ul>
<p>Näkymät tallennetaan tietokannan istuntotilan mukana ja palautetaan sen uudelleenavauksen yhteydessä.</p>
<h3>Asetukset</h3>
<p>Työkalurivin asetuspainike (rataskuvake), ohjepainikkeen vasemmalla puolella, avaa blunderDB:n asetusikkunan. Se on jaettu kuuteen välilehteen:</p>
<ul>
<li><strong>Käyttöliittymä</strong> — kieli, näytön skaalaus, paneelin sijainti;</li>
<li><strong>Laudan värit</strong> — laudan värit;</li>
<li><strong>Bearoff</strong> — Eval-paneelin käyttämät ulosmenotaulukot;</li>
<li><strong>gammonNet</strong> — sisäänrakennetun evaluaattorin asetukset, kuvattu alla;</li>
<li><strong>Valvottu kansio</strong> — kansioon saapuvien otteluiden automaattinen tuonti, kuvattu alla;</li>
<li><strong>Merkitsijän identiteetti</strong> — avain, jolla alkuperämerkintäsi allekirjoitetaan; kuvattu luvussa Tietokannan jakaminen: alkuperä ja salasana.</li>
</ul>
<p><em>Käyttöliittymä</em>-välilehti alkaa <strong>teemalla</strong>: <em>seuraa järjestelmää</em>, <em>vaalea</em>, <em>tumma</em>, <em>suuri kontrasti</em> tai <em>tulostettava</em>. Teema asettaa käyttöliittymän värit ja <strong>ehdottaa lautapalettia</strong> — tumma käyttöliittymä vaalean laudan ympärillä ei ole tumma teema vaan puolikas, sillä lauta täyttää suurimman osan ikkunasta.</p>
<p>Sinulla on viimeinen sana, ja mekanismi takaa sen sen sijaan että lupaisi: <em>Värit</em>-välilehti säätää edelleen lautaa suoraan, ja teeman jälkeen valittu väri on sinun. Käynnistyksessä sovelletaan vain käyttöliittymän symboleja, ei koskaan lautapalettia — asettamasi on jo ladattu, ja sen ylikirjoittaminen joka käynnistyksessä pyyhkisi työsi istunto kerrallaan. Katso <code>ADR-0038 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0038-a-named-theme-carries-the-board-palette-and-the-user-still-has-the-last-word.md&gt;</code>__.</p>
<p><em>Seuraa järjestelmää</em> on oletus: se noudattaa työpöydän vaalea/tumma-asetusta, myös kun se muuttuu kesken istunnon. Työkalu ei tyrkytä vaaleaansa tai tummaansa työpöydälle joka on jo päättänyt.</p>
<p><em>Käyttöliittymä</em>-välilehdellä voi myös valita kielen: englanti, ranska, saksa, italia, espanja, suomi, japani, kreikka tai venäjä. Koko käyttöliittymä (työkalurivi, paneelit, viestit, ohje) käännetään valitulle kielelle. Kielivalinta tallennetaan ja säilyy istunnosta toiseen.</p>
<p>Samalta välilehdeltä löytyy myös painike <strong>Tiivistä tietokanta</strong>, joka ottaa takaisin poistojen (ottelut, turnaukset, siivoukset) jättämän levytilan: tietokanta ei koskaan pienene itsestään dataa poistettaessa, tiivistys on pyydettävä nimenomaisesti. Toiminto voi kestää suuressa tietokannassa ja vaatii tilapäisesti noin kaksinkertaisen koon verran vapaata levytilaa (blunderDB kieltäytyy käynnistymästä sen sijaan, että riskeeraisi keskeytyneen tiivistyksen); siksi ennen käynnistystä pyydetään vahvistus. Tulos — säästynyt tila megatavuina — näkyy sen jälkeen tilarivillä. Sama toiminto on käytettävissä komentoriviltä komennolla <code>blunderdb vacuum</code> (katso Komentoriviliittymä (CLI)).</p>
<p>Sen alapuolella oleva <strong>Avaa lokikansio</strong> -painike avaa kansion, jossa sovelluksen loki sijaitsee — kätevää, kun vikailmoitukseen halutaan liittää yksityiskohtia, erityisesti kun blunderDB on käynnistetty pikakuvakkeesta tai kaksoisnapsautuksella ilman päätettä, joka näyttäisi mitään.</p>
<p>Oletuksena pois päältä oleva <strong>Tarkista päivitykset käynnistyksessä</strong> -valintaruutu kysyy kerran käynnistystä kohden GitHub-arkiston julkaisusivulta ja näyttää tilarivillä viestin, jos uudempi versio on saatavilla — ei koskaan ikkunaa, joka estäisi työskentelyn. Tarkistus pysyy automaattisesti pois päältä asennuksessa, joka on tehty paketinhallinnan kautta (Flatpak, Homebrew, jakelun paketti…): silloin päivityksistä huolehtii se kanava eikä blunderDB itse.</p>
<p><em>Laudan värit</em> -välilehdellä voi mukauttaa laudan värejä. Jokaisella osalla on oma värivalitsimensa: tausta, reunus, vaaleat ja tummat kolmiot, pelaajan 1 ja pelaajan 2 nappulat, nopat, noppien silmäluvut ja tuplauskuutio. <em>Palauta</em>-painike palauttaa kaikki oletusvärit. Kielen tavoin valitut värit säilyvät istunnosta toiseen.</p>
<p><em>Bearoff</em>-välilehti hallitsee Eval-paneelin ulosmenotaulukoita (ks. Eval-paneeli). Niitä <strong>ei ole upotettu ohjelmatiedostoon eikä niitä ladata</strong>: blunderDB laskee ne koneella, joka niitä käyttää, ja tulos on tavu tavulta sama kuin gnubg:n tuottama — SHA-256-tiiviste tarkistetaan ennen kuin taulukko hyväksytään.</p>
<p>Kaksi tavallista taulukkoa (TS-06-06 tuplausratkaisulle, OS-06 EPC:lle) lasketaan ensimmäisellä käynnistyksellä taustalla ja kysymättä: noin kuusi sekuntia yhdellä ytimellä, joiden aikana sovellusta käytetään normaalisti. Eval-paneeli mainitsee siitä vain, jos siihen asetetaan asema, joka tarvitsee vielä valmistumatonta taulukkoa.</p>
<p>Välilehti näyttää aktiivisen alueen ja sen alkuperän, EPC:n lukeman yksipuolisen taulukon tilan, kansion jossa kaikki tämä sijaitsee, ja luettelon olemassa olevista taulukoista kokoineen ja tuomioineen. Jokainen rivi poistetaan erikseen, vahvistuksen jälkeen.</p>
<p><strong>Varmennettu vai varmentamaton.</strong> <em>Varmennetulla</em> taulukolla on täsmälleen ne tavut, jotka gnubg tuottaa sen alueelle: sen SHA-256-sormenjälki on blunderDB:ssä ja se löytyi uudelleen. Yksipuolisille taulukoille (OS-06–OS-10) tallennetut sormenjäljet ovat ne, jotka GNUbg 1.08:n <code>makebearoff</code>-työkalu tuottaa. <em>Varmentamaton</em> taulukko on hyvin muodostettu, mutta sen alueelle ei ole tallennettua sormenjälkeä — sitä ei moitita mistään, kukaan ei vain ole verrannut sitä viitteeseen. <em>Vioittunut</em> taulukko on ristiriidassa itsensä kanssa eikä sitä koskaan lueta; se lasketaan uudelleen.</p>
<p><strong>Laajemman taulukon laskeminen.</strong> Alue valitaan kahden perheen luettelosta yhdessä sille annettavien ytimien määrän kanssa (oletuksena kaikki paitsi yksi, jotta kone pysyy käytettävänä):</p>
<ul>
<li><strong>tarkka kuutio (kaksipuolinen)</strong>, TS-06-06:sta TS-06-15:een: laajentaa aluetta, jolla voittotodennäköisyys ja kuution tuomio luetaan eikä arvioida;</li>
<li><strong>EPC kotialueen ulkopuolella (yksipuolinen)</strong>, OS-06:sta OS-10:een: laajentaa sitä, kuinka kaukana kotoa nappula saa olla ilman että EPC-lohko vaikenee. Tämä ajo lukee vain laskettavaa pienempiä asemia, joten se on rakenteeltaan peräkkäinen eikä ydinmäärä hyödytä sitä — valitsin sanoo sen harmaantumalla.</li>
</ul>
<p>Ennen kuin mitään aloitetaan, välilehti kertoo valitulle alueelle kolme lukua: koon levyllä, laskennan aikana tarvittavan muistin ja ajan, jonka sen pitäisi viedä <em>tällä koneella</em>. Viimeksi mainittu alkaa arviona ja muuttuu mittaukseksi: jokainen riittävän laaja ajo kirjaa oman nopeutensa ja säilyttää sen. Alue, jota käytettävissä oleva muisti ei salli, tarjotaan harmaana ja perusteluineen — ”tarvittaisiin 24 Gt, jäljellä on 12” on vastaus, puuttuva rivi ei olisi.</p>
<p>Suuruusluokkana kuudentoista säikeen koneella: TS-06-09 painaa 191 Mt ja vie kymmenkunta sekuntia, TS-06-11 painaa 1,2 Gt ja muutaman minuutin, TS-06-13 ylittää sen mitä useimmat koneet pystyvät pitämään muistissa. Yksipuolisella puolella, yhdellä ytimellä: OS-07 painaa 4,9 Mt ja vie 17 s, OS-08 15 Mt ja 1 min 20, OS-10 117 Mt ja puoli tuntia.</p>
<p><strong>Tauko ja jatkaminen.</strong> Laskennan aikana edistyminen näyttää <em>mitatun</em> jäljellä olevan ajan ja kaksi erillistä painiketta: <em>Tauko</em> ja <em>Peruuta</em>. Tauko kirjoittaa laskennan tilan taulukon viereen; uudelleen käynnistäminen jatkaa siitä mihin jäätiin sen sijaan että aloitettaisiin alusta. Peruuttaminen ei säilytä mitään. Asetusikkunan sulkeminen ei keskeytä mitään — laskenta jatkuu taustalla.</p>
<p>Tauolle jätetty laskenta löytyy seuraavalta käynnistykseltä, nimettynä ja lukuineen (”TS-06-09 keskeytyi kohdassa 43 %”), painikkeineen <em>Jatka</em> ja <em>Poista</em>. Mikään ei käynnisty itsestään uudelleen: käyttäjä pyysi pysäytystä.</p>
<p>Välilehti sallii lopuksi osoittaa ulkoiseen kaksipuoliseen <code>.bd</code>-tiedostoon, esimerkiksi gnubg:n itsensä tuottamaan tietokantaan: laajimman alueen taulukko voittaa.</p>
<p><em>Yleiset</em>-välilehti kantaa lopuksi <strong>Korjaa analyysit</strong>: analyysisarakkeet, joita haku ja tilastot kysyvät, ovat projektio tallennetuista analyyseista, jotka pysyvät koskemattomina. Projektion vika on siis korjattavissa ilman uudelleentuontia. Se on nimenomaista eikä koskaan automaattista — jonkun analyysisarakkeiden uudelleenkirjoittaminen pelkästään siksi, että hän avaa tietokantansa, ei ole asia jonka työkalun tulisi tehdä hänen selkänsä takana. Sama <code>blunderdb repair</code> on käytettävissä komentoriviltä.</p>
<p><strong>gammonNet</strong>-välilehti säätää sisäänrakennettua evaluaattoria (katso <code>ADR-0011 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0011-gammonnet-is-ported-to-go-and-the-representation-boundary-sits-at-the-evaluator-s-edge.md&gt;</code>__). Siinä on kaksi säädettävää hakusyvyyttä, jotka on nimetty ja tallennetaan erikseen — toisen alentaminen ei koskaan muuta toista:</p>
<ul>
<li><strong>Näyttösyvyys</strong> — interaktiivinen mukavuus lautaa muokattaessa; ei koskaan kirjoiteta tietokantaan.</li>
<li><strong>Analyysisyvyys</strong> — se, mitä tuonnin jälkeinen analyysierä kirjoittaa aseman Analyysiin.</li>
</ul>
<p>Molempien oletusarvo on <strong>2-ply</strong>, kanoninen asetus. Välilehdellä voi säätää myös <strong>karsinnan</strong> (oletus <code>k=12</code>) ja <strong>näytettävien ehdokassiirtojen määrän</strong> (oletus 10) sekä valintaruudun <strong>analysoi automaattisesti tuonnin jälkeen</strong>, joka käyttöön otettuna tarkistaa jokaisen tuonnin jälkeen, onko jäljellä asemia <strong>ilman yhtään analyysiä</strong> (ei gammonNet-, XG-, GNUbg- eikä BGBlitz-analyysiä — sääntö on « evaluointi täyttää vain aukon », ei koskaan korvaa), ja tarvittaessa käynnistää taustalla gammonNet-analyysin määritetyllä analyysisyvyydellä. Painike <strong>Analysoi nyt</strong> käynnistää saman täydennysanalyysin uudelleen manuaalisesti — hyödyllinen ennen tätä ominaisuutta luodun kirjaston saattamiseksi ajan tasalle.</p>
<p>Toinen painike, <strong>Analysoi vanhentuneet positiot uudelleen</strong>, kattaa päinvastaisen tapauksen: jo gammonNetin analysoima positio, jonka tallennettu analyysi on kuitenkin kirjoitettu nyt käynnissä olevaa vanhemmalla moottoriversiolla tai eri syvyydellä kuin yllä määritetty analyysisyvyys, merkitään siellä vanhentuneeksi ja arvioidaan uudelleen. Positiota, jolla on lisäksi XG-, GNUbg- tai BGBlitz-analyysi, tämä painike ei koskaan koske, riippumatta sen gammonNet-sisällöstä — <code>ADR-0013 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0013-evaluations-fill-gaps-an-imported-analysis-is-never-overwritten.md&gt;</code>__:n suoja pysyy ehdottomana. Kunkin painikkeen vieressä näkyvä luku (asemat ilman analyysiä, vanhentuneet asemat) on puhtaasti informatiivinen; erä laskee oman listansa uudelleen käynnistyessään.</p>
<p>Molemmat erät ovat <strong>rajattuja, näkyviä ja peruutettavissa, eivät koskaan hiljainen daemon</strong>: niiden edistyminen (<code>analysoidut asemat / yhteensä</code>) ja peruutuspainike näkyvät tilarivillä koko niiden keston ajan ja katoavat, kun ne on valmis, ja niiden tilalle tulee tuloksen yhteenveto — kuinka monta positiota <strong>analysoitiin</strong>, kuinka monta <strong>hylättiin</strong> (positio, jota gammonNet kieltäytyy arvioimasta, kuten sen taulukon kattavuuden ulkopuolella oleva ottelutulos, mikä ei koskaan ole virhe) ja kuinka monta <strong>epäonnistui</strong> (yritetään uudelleen muuttumattomana seuraavalla ajolla). Sovelluksen sulkeminen kummankaan aikana ei hukkaa mitään: jokainen analysoitu positio kirjoitetaan sitä mukaa, ja seuraava ajo jatkaa täsmälleen siitä, mihin analyysi jäi, ilman minkäänlaista lokia.</p>
<p><strong>Ilman analyysiä tuotu ottelu saa näin PR-luvun.</strong> Näin on verkossa pelatun ottelun tai Jellyfish-<code>.mat</code>-tiedoston laita, kun kukaan ei ole ajanut sitä XG:n läpi: blunderDB tunsi asemat ja pelatut siirrot, mutta mikään analyysi ei kertonut mitä ne olivat arvoltaan. Erän ajon jälkeen todella pelattua siirtoa verrataan gammonNetin järjestykseen, ja ero syöttää PR:n, virheprosentin, pahimmat päätökset ja kaikki muut mittarit, aivan kuten XG:n analysoimassa ottelussa. Vertailu ei keksi mitään: pelattu siirto tulee ottelun omasta siirtotaulusta, joka kirjoitetaan tuonnissa riippumatta siitä kantoiko tiedosto analyysin.</p>
<p>Tätä vanhemmalla versiolla analysoitua tietokantaa ei tarvitse arvioida uudelleen: <code>blunderdb repair</code> laskee sarakkeet uudelleen jo tallennetuista analyyseistä ja siirroista ja palauttaa noille otteluille niiden PR:n (katso repair).</p>
<p>Rehellinen varaus: asema tunnistetaan rakenteestaan, joten kahdesti kohdattu asema — kerran hyvin, kerran huonosti pelattu — kantaa vain yhden eron, ensimmäisen kirjatun esiintymänsä eron. Tämä ei ole tälle laskennalle ominaista: XG-kirjastolla on täsmälleen sama muoto.</p>
<h4>Valvottu kansio</h4>
<p><strong>Valvottu kansio</strong> -välilehti pyytää blunderDB:tä katsomaan kansiota käydessään ja tuomaan jokaisen ottelutiedoston, joka siihen <strong>ilmestyy</strong>. Pelaat istunnon eXtreme Gammonissa, palaat blunderDB:hen, ja ottelut ovat jo siellä.</p>
<p>Mitään ei arvata. Ennen kuin kansio on nimetty, valvontaa ei ole: blunderDB ei ryhdy lukemaan hakemistoa siksi, että se arveli mistä ottelusi löytyvät. <strong>Ehdota</strong>-painike katsoo tämän koneen tavanomaisia paikkoja ja tarjoaa jonkin vain jos se todella on olemassa; muuten se sanoo niin, ja kansion nimeäminen jää sinulle.</p>
<p>Kolme asiaa kannattaa tietää ennen ruudun rastittamista:</p>
<ul>
<li><strong>Vain ilmestyvät tiedostot tuodaan.</strong> Se mitä kansiossa jo on valvonnan alkaessa kirjataan tunnetuksi ja jätetään rauhaan: valvonnan osoittaminen neljän vuoden otteluihin ei saa tuoda niitä kaikkia. Paikalla olevan tuomiseen on kansion tuonti, joka on sitä varten — ja nämä kaksi sopivat hyvin yhteen: ensin tuonti, sitten valvonta.</li>
<li><strong>Tiedosto tuodaan vasta kun sen koko on vakiintunut.</strong> Ottelu, jota toinen ohjelma kirjoittaa, kasvaa vilkaisusta toiseen; puoliksi kirjoitettuna tuotuna siitä tulisi jäsennysvirhe, jolle kukaan ei voi mitään. blunderDB odottaa siis näkevänsä saman tiedoston kahdesti muuttumattomana.</li>
<li><strong>Tuonti on hiljainen.</strong> Olit tutkimassa asemaa kun ottelusi saapuivat: ruudun viemisestä sinulta olisi pahin mahdollinen hetki. Tuonti tapahtuu ilman ikkunaa, ja tilarivillä näkyy palkki, joka kertoo tuotujen, ohitettujen (kaksoiskappaleet) ja epäonnistuneiden otteluiden määrän, sekä painike joka avaa halutessa koko raportin. Kaikki muu on samaa kuin käsin tehdyssä tuonnissa: samat kaksoiskappaleet havaittuina, sama tuontierä, sama automaattinen analyysi jos se on päällä.</li>
</ul>
<p>Oletusväli on kymmenen sekuntia; alaraja kaksi. Kansiota ei käydä läpi rekursiivisesti: valvottu kansio on paikka johon työkalu pudottaa ottelunsa, ei tutkittava puu. Irrotettu verkkojako ei pysäytä valvontaa eikä saa sisältöään käymään uudesta palatessaan.</p>
<p>Sama valvonta on olemassa komentorivillä komennolla <code>blunderdb import --type batch --dir &lt;kansio&gt; --watch</code> (katso Komentoriviliittymä (CLI)): se on muoto, jota palvelin, ajastettu tehtävä tai skripti voi käyttää.</p>
<p>Asetusikkuna sisältää myös käyttöliittymän näyttöasetukset. <strong>Käyttöliittymän skaalaus</strong> -liukusäätimellä voi suurentaa tai pienentää kaikkia käyttöliittymän elementtejä, mistä on hyötyä korkean tarkkuuden näytöillä tai luettavuuden parantamiseksi. <strong>Paneelien sijainti</strong> -valikko määrittää, missä paneelit (haku, ottelut, analyysi) näkyvät suhteessa lautaan: <em>alhaalla</em>, <em>sivulla</em> tai <em>automaattinen</em> (sivu valitaan tällöin leveillä näytöillä käytettävissä olevan tilan parempaa hyödyntämistä varten). Muiden asetusten tavoin nämä valinnat säilyvät istunnosta toiseen.</p>
<h3>Opastetut kierrokset ja esimerkkitietokanta</h3>
<p>Käytön aloittamisen helpottamiseksi blunderDB tarjoaa käyttöliittymästä <strong>opastettuja kierroksia</strong>. Kierrosten luettelo avataan työkalupalkista tai komennolla <code>tour</code> (alias <code>tutorial</code>). Käytettävissä on seitsemän kierrosta: yleinen käyttöliittymäkierros sekä asemien hakuun, otteluiden tarkasteluun, turnausten tarkasteluun, Eval-paneeliin, Anki-kertaukseen ja tilastoihin keskittyvät kierrokset. Jokainen kierros korostaa käyttöliittymän asianmukaiset elementit vaihe vaiheelta, avaa matkan varrella sen paneelin josta se puhuu, ja sen voi toistaa milloin tahansa. Ensimmäisellä käynnistyskerralla yleinen kierros tarjotaan automaattisesti.</p>
<p>Komento <code>demo</code> lataa <strong>esimerkkitietokannan</strong>, jonka avulla voit tutustua työkalun ominaisuuksiin tuomatta omia pelejäsi: kolme ottelua (joista kaksi on koottu turnaukseksi), jotka eXtreme Gammon, BGBlitz ja gammonNet ovat analysoineet, kolme aihekohtaista kokoelmaa, tunnisteilla merkittyjä kommentteja (<code>#blunder</code>, <code>#cube</code>) sekä Anki-pakka kertauslokeineen. Pelaajat, turnaus ja paikka ovat keksittyjä. Opastetut kierrokset käyttävät tätä tietokantaa, kun mitään tietokantaa ei ole avattu.</p>
<h3>Asemien selaaminen</h3>
<p>Oletuksena blunderDB mahdollistaa seuraavat:</p>
<ul>
<li>selata nykyisen kirjaston eri asemia — sitä ei koskaan ladata yhtenä möhkäleenä: blunderDB pitää siitä vain tunnisteiden listaa ja lataa asemat viidenkymmenen ikkunoissa näytettävän aseman ympäriltä, joten kymmenientuhansien asemien tietokanta aukeaa yhtä nopeasti kuin pienikin,</li>
<li>näyttää asemaan liittyvät analyysitiedot,</li>
<li>näyttää, lisätä ja muokata aseman kommentteja.</li>
</ul>
<p>Työkalurivin painike <strong>Siirry asemaan</strong> avaa ikkunan, johon voi kirjoittaa suoraan aseman järjestysnumeron ja hypätä siihen ilman vierittämistä. Se on komentorivin <code>[number]</code>-komennon graafinen vastine (katso Asemat ja navigointi).</p>
<div class="admonition tip">
<p>Katso saatavilla olevat pikanäppäimet kohdasta Näppäimistöoikotiet.</p>
</div>
<h3>Asemien muokkaus</h3>
<p><em>TAB</em>-näppäimen painaminen avaa hakupaneelin ja mahdollistaa aseman muokkaamisen laudalla sen lisäämiseksi tietokantaan tai haettavan asemarakenteen määrittämiseksi. Pelinappuloiden, kuution, pistetilanteen ja vuoron jakaumaa voi muokata hiirellä (katso Muokkaa asemaa).</p>
<div class="admonition tip">
<p>Katso saatavilla olevat pikanäppäimet kohdasta Näppäimistöoikotiet.</p>
</div>
<h3>Komentorivi</h3>
<p>Tilariviin upotettu komentorivi mahdollistaa kaikkien graafisessa käyttöliittymässä saatavilla olevien blunderDB:n toimintojen suorittamisen: yleiset tietokantatoiminnot, asemissa navigointi, analyysin ja/tai kommenttien näyttäminen, asemien haku suodattimilla... Kun käyttöliittymä on tullut tutuksi, suositellaan vähitellen siirtymistä komentorivin käyttöön, joka mahdollistaa blunderDB:n tehokkaan ja sujuvan käytön, erityisesti asemahakutoiminnoissa.</p>
<p>Avaa komentorivi painamalla <em>VÄLILYÖNTI</em>-näppäintä. Lähetä kysely ja sulje komentorivi painamalla <em>ENTER</em>-näppäintä.</p>
<p>blunderDB suorittaa käyttäjän lähettämät kyselyt edellyttäen, että ne ovat kelvollisia, ja muuttaa tarvittaessa tietokannan tilaa välittömästi. Käyttäjältä ei vaadita erillisiä tallennustoimia.</p>
<div class="admonition tip">
<p>Katso komentorivillä käytettävissä olevien komentojen luettelo kohdasta komentoluettelo.</p>
</div>
<h3>Analyysipaneeli</h3>
<p><strong>Analyysipaneeli</strong> (<em>CTRL-L</em>) näyttää nykyisen aseman analyysitiedot, jotka on tuotu lähteistä eXtreme Gammon (XG), GNUbg tai BGBlitz. Se esittää parhaat vaihtoehdot (pelinappulasiirrot tai kuutiopäätökset) niiden ekviteettiarvoineen ja vastaavine virheineen. <em>d</em>-näppäin vaihtaa pelinappulasiirtojen analyysin ja kuutioanalyysin välillä. Ottelussa navigoitaessa todella pelattu siirto korostetaan vaihtoehtojen luettelossa. Näytä tai piilota paneeli painamalla <em>CTRL-L</em> tai suorittamalla komento <code>list</code>.</p>
<p>Taulukoiden alla <strong>lause</strong> kertoo joskus, mitä pelattu päätös maksoi ja miksi: ”Menetät 120 mMWC: pelattu siirto jättää kolme yksinäistä nappulaa, kun 13/7 8/7 jättää vain yhden.” Se syntyy kuudesta mitattavasta säännöstä — alttiudesta, tehdystä tai menetetystä kotipisteestä, luovutetuista gammon-mahdollisuuksista, turvallisuudesta joka maksaa enemmän kuin tuottaa, ja kuutiovirheen kahdesta suunnasta (tuplaus liian myöhään tai liian aikaisin, liian löysä hyväksyntä tai liian tiukka luovutus).</p>
<p>Tärkein sääntö on <strong>vaikeneminen</strong>: lause ilmestyy vain, kun sääntö pätee varmasti, ja virheestä joka ylittää kynnyksen, josta lähtien moottorit ovat yhtä mieltä siitä että kyseessä on virhe. Muulloin lausetta ei ole — ei tyhjää kehystä eikä ”emme tiedä”. Väärä selitys maksaa enemmän kuin ei selitystä: se opettaa jotain epätarkkaa.</p>
<p>Kun asemaa on arvioinut <strong>useampi moottori</strong>, paneelin yläreunan palkki asettaa ne rinnakkain: yksi rivi moottoria kohden, sen syvyys ja sen vastaus — kuutiotuomio tai sen oma paras siirto. Se kertoo ensin ovatko ne samaa mieltä, ja juuri erimielisyys sen oikeuttaa: ”XG sanoo tuplaus, otto; gammonNet sanoo ei tuplausta” luetaan yhdellä silmäyksellä, siinä missä ennen piti verrata kahta taulukkoa vinottain.</p>
<p>Moottorin paras siirto on <strong>sen moottorin</strong> paras: ehdokaslista on lajiteltu ekviteetin mukaan kaikki moottorit sekaisin, joten sen ensimmäinen alkio ei ole kenenkään paras siirto erityisesti.</p>
<p>Palkki ilmestyy vain kun moottoreita todella on useampi, ja se on olemassa vain tässä paneelissa: Eval-paneeli esittää <strong>yhden</strong> päätöksen, sisäänrakennetun moottorin (<code>ADR-0017 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0017-the-panel-shows-position-facts-plus-the-one-decision-the-board-asks.md&gt;</code>__), eikä vertailulla olisi siellä sijaa.</p>
<p>Siirrot kirjoitetaan niin kuin ne luetaan laudalta, täällä kuten Eval-paneelissakin: vähiten edennyt nappula liikkuu ensin, ja <strong>nappula, joka käyttää useamman nopan peräkkäin, kirjoitetaan vain kerran</strong> — samalla nappulalla pelattu 64 luetaan <code>24/14</code>, ja <code>24/14*</code>, jos se lyö perille tullessaan. Ketjun yksityiskohdat näkyvät vain silloin, kun ne kertovat jotakin lisää: <em>matkan varrella</em> tehty lyönti säilyttää välipisteensä, <code>24/18* 18/14</code>, muuten lyönti pisteessä 18 katoaisi merkinnästä.</p>
<p>Tuodun analyysin ekviteetti noudattaa samaa sääntöä kuin Eval-paneeli: sarake ilmoittaa oman viitekehyksensä, ”Equity (money)” tai ”Equity (match)” analysoidun aseman pistetilanteen mukaan, ei koskaan pelkkää ”Equity”-sanaa kertomatta asteikkoa. Money game -asemassa voimassa olevat <strong>Jacoby</strong>- ja <strong>Beaver</strong>-säännöt näytetään myös pieninä merkkeinä kuutiopäätöstaulukon alla.</p>
<h3>Kommenttipaneeli</h3>
<p><strong>Kommentit</strong>-paneeli (<em>CTRL-P</em>) näyttää, lisää ja muokkaa nykyiseen asemaan liitettyjä kommentteja. Asemalla voi olla useita: kaikki näytetään, uusimmasta vanhimpaan. XG-tiedostoista tuodut kommentit liitetään automaattisesti vastaaviin asemiin. Paina <em>CTRL-P</em> tai suorita komento <code>comment</code> näyttääksesi tai piilottaaksesi paneelin.</p>
<p>Jokainen tiedostosta tullut kommentti kantaa <strong>alkuperämerkintää</strong> (<code>XG</code>, <code>GNU BG</code>, <code>BGF</code>, tai <em>tuotu</em>, kun alkuperää ei koskaan tallennettu). Itse kirjoittamasi kommentit eivät kanna sellaista: se on tavallinen tapaus, ja jokaisen rivin merkitseminen olisi vain kohinaa. Tuodun kommentin muokkaaminen tekee siitä sinun: muokkauksen jälkeen lause on sinun.</p>
<p>Ero näkyy muuallakin: ottelun poistaminen ei enää tuhoa asemaa, johon <strong>sinä</strong> olit kirjoittanut. Lähdetiedostosta poimittu muistiinpano sen sijaan katoaa yhä sen ottelun mukana, joka sen toi.</p>
<h4>Tunnisteet</h4>
<p><strong>Tunniste</strong> on kommenttiin kirjoitettu <code>#sana</code>. Mikään ei ilmoita sitä, mikään taulu ei säilytä sitä, ja se on tarkoituksellista: sanasto on sinun omaa proosaasi, ja ilmoituksen vaatiminen ennen tunnisteen käyttöä muuttaisi tavan paperityöksi.</p>
<p>Puuttui toinen puoli: <strong>nähdä</strong> se sanasto, jonka on itselleen rakentanut, ja napsauttaa tunnistetta sen sijaan että muistelisi miten sen kirjoitti. Komento <code>tags</code> tai kirjoituskentän vieressä oleva <code>#</code>-painike avaa sanastoikkunan: tämän tietokannan tunnisteet, kukin sitä kantavien <strong>asemien määrän</strong> kanssa, napsautettavina vastaavan haun käynnistämiseksi. Listan alla ovat suositellut tunnisteet, joita tämä tietokanta ei vielä käytä — backgammon-kirjallisuudesta poimittu sanasto (<code>#blitz</code>, <code>#prime</code>, <code>#holding</code>, <code>#backgame</code>, <code>#containment</code>, <code>#crunch</code>, <code>#ace-point</code>, <code>#timing</code>…), ehdotettu eikä koskaan pakotettu: listalta puuttuva tunniste on täsmälleen yhtä arvokas kuin listalla oleva.</p>
<p>Kirjoittaessa <code>#</code> ehdottaa niitä tunnisteita, joita <strong>tämä tietokanta</strong> jo käyttää, ja sitten suositeltuja. Juuri se estää kirjoittamasta <code>#back-game</code> yhtenä päivänä ja <code>#backgame</code> seuraavana — mitä mikään muu ei huomaisi.</p>
<p>Tunnistehaku kirjoitetaan komentorivillä muodossa <code>#prime</code>. Se on <strong>rajattu</strong>: <code>#prime</code> ei löydä sanaa <code>#priming</code>, kun taas tavallinen tekstihaku, joka etsii osamerkkijonoa, ei osaa erottaa niitä. Useat tunnisteet <strong>kasautuvat</strong> — <code>s #prime #backgame</code> pyytää asemat, jotka kantavat molemmat — koska asema kantaa useita tunnisteita: kahden nimeäminen voi tarkoittaa vain ”molempia”. Tämä on päinvastoin kuin vaihe- tai alkuperäsuodattimessa, jossa asemalla on vain yksi arvo ja kahden arvon nimeäminen voi tarkoittaa vain ”jompaakumpaa”.</p>
<p>Sama lista saadaan käyttöliittymän ulkopuolella komennolla <code>blunderdb list --type tags</code> (katso Komentoriviliittymä (CLI)).</p>
<h3>Roskakori</h3>
<p>Aseman, kokoelman tai kommentin poisto kulkee nyt <strong>roskakorin</strong> kautta: poisto todella tapahtuu, mutta kopio katoavasta säilytetään kolmekymmentä päivää. Komento <code>trash</code> avaa ikkunan, joka listaa ne, kussakin <em>Palauta</em> ja <em>Poista</em>.</p>
<p>Palautettu asema tulee takaisin <strong>analyyseineen ja kommentteineen</strong> — sen palauttaminen alastomana olisi palautus vain nimeltään. Se ei palaa vanhalla numerollaan: alkuperäistä riviä ei enää ole, ja blunderDB tallentaa sen uudelleen sormenjälkensä perusteella, mikä takaa ettei kaksoiskappaletta synny mutta antaa sille uuden tunnisteen. Kokoelma palaa listoineen; sen sisältämiä asemia ei koskaan poistettu — kokoelma on näkymä niihin.</p>
<p>Yli kolmekymmentä päivää vanhat poistaa komento <code>vacuum</code>, ei koskaan tietokannan avaaminen: <code>vacuum</code>:in tekemättä jättäminen on kaiken säilyttämistä.</p>
<div class="admonition note">
<p>Roskakori ei matkusta. Vienti ei kanna sitä mukanaan, eikä ottelun poisto laita siihen mitään: ottelun poistoa seuraava orpojen asemien siivous on automaattista siivousta eikä käyttäjän teko — katso säilytyssääntö kohdassa Ottelupaneeli.</p>
</div>
<h3>Hakupaneeli</h3>
<p><strong>Hakupaneeli</strong> (<em>CTRL-F</em> tai <em>TAB</em>) suodattaa asemia vapaasti yhdisteltävien kriteerien mukaan: pelinappularakenne, kuutiopäätöksen tyyppi, virheen suuruus, päivämäärät, tunnisteet jne. <em>TAB</em>-näppäin avaa samanaikaisesti hakupaneelin ja asemaeditorin, jolloin haettava pelinappularakenne voidaan määrittää suoraan laudalla.</p>
<p>Tarkenna hakua tällä hetkellä suodatettujen asemien joukossa käyttämällä komentoa <code>ss</code>, jota seuraavat suodattimet (esim. <code>ss nc</code>, <code>ss E&gt;40</code>). Hakupaneeli tarjoaa samaa toimintoa varten myös valintaruudun <em>Hae nykyisistä tuloksista</em>.</p>
<p>Paneeli tarjoaa nimenomaisen hallinnan haettavalle <strong>päätöstyypille</strong>: <em>Indifférent</em> (ei suodatinta), <em>Siirto</em> (siirtopäätökset) tai <em>Tuplaus</em> (kuutiopäätökset). Kun <em>Tuplaus</em> on valittuna, toinen luettelo tarkentaa alityypin: <em>Kaikki</em>, <em>Tuplaus / Ei tuplausta</em> (vuorossa olevan pelaajan on päätettävä, tuplaako) tai <em>Hyväksy / Luovuta</em> (vastaus vastustajan tuplaukseen). Hallinta on synkronoitu laudan kanssa: noppien tai kuution muuttaminen laudalla päivittää päätöstyypin ja päinvastoin. <em>Hyväksy / Luovuta</em> -tilassa kuutio näytetään laudan keskellä tarjotulla arvolla; tämä arvo on edelleen muokattavissa.</p>
<p><strong>Pelin vaihe</strong> — avaus, keskipeli, kilpajuoksu, nappuloiden poisto — on merkintä, jonka blunderDB laskee pelkästä laudasta. Sitä ei voi koskaan muokata, ja se on haettavissa komentorivin <code>ph:</code>-merkinnällä (<code>ph:race</code>, toistettavissa: <code>ph:race ph:bearoff</code>). Kolme sen neljästä rajasta ovat ne, joilla GNU Backgammon ohjaa verkkojaan; neljäs, jossa avaus päättyy, on blunderDB:n käytäntö: asema on yhä avauksessa niin kauan kuin kumpikaan puoli ei ole siirtänyt yli neljää nappulaa lähtöpisteiltään, mitään ei ole poistettu eikä mikään ole palkissa.</p>
<div class="admonition note">
<p>Merkinnän laskee uudelleen komento <code>blunderdb repair</code>. Tietokannassa, joka avataan ensimmäistä kertaa tällä versiolla, laskenta tehdään kerran, avattaessa. Tietokanta, jonka vaiheita ei ole koskaan laskettu, ei palauta mitään <code>ph:</code>-haulle — ei mitään, väärän vastauksen sijaan.</p>
</div>
<p>Komento <code>like</code> vastaa eri kysymykseen kuin tunnukset: se korvaa selatun listan nykyistä <strong>lähimmillä</strong> asemilla, lähimmästä kaukaisimpaan. Läheisyys on kuljetusetäisyys nappulapipeinä — se määrä nappuloiden liikettä, joka erottaa asemat — ja näkökulma on aina vuorossa olevan pelaajan. Se ei ole suodatin: samankaltaisuus <strong>järjestää</strong> koko kirjaston sen sijaan että rajaisi sitä, eikä siksi yhdisty tunnuksiin.</p>
<p>Tunnus <code>n</code> laskee <strong>kohtaamisia</strong>: <code>n&gt;3</code> säilyttää asemat, joihin johtaa yli kolme siirtoa, kaikissa otteluissa. Se on eri kysymys kuin ”missä menin vikaan” — kaksikymmentä kertaa kohdattu ja yhdeksäntoista kertaa oikein pelattu asema on yhä se, joka pitää osata ulkoa. Lasketaan siirrot, ei otteluita: sama asema kahdesti yhdessä ottelussa on kaksi, koska ne olivat kaksi päätöstä.</p>
<p>Sanallinen lause voi korvata tunnukset <code>ask</code>-komennolla: <code>ask my cube blunders at a score</code>. Lause <strong>käännetään tunnuksiksi</strong>, jotka kirjoitetaan komentoriville — lue ne ja aja sitten. Mitään ei arvata eikä mikään lähde koneelta: sanasto on kiinteä, sama lause antaa aina saman kyselyn, ja se mitä ei ymmärretty <strong>sanotaan</strong> eikä sivuuteta. Väärä käännös näkyy siis ennen kuin se palauttaa vääriä tuloksia, ja tunnukset oppii lukemalla ne.</p>
<p>Kaksi aikomusta eivät ole tunnuksia ja asetetaan hakulaudalle rivin sijaan: ”kuutio” tai ”siirto” (päätöksen laji) ja ”ottelutilanteessa” tai ”money”. <code>ask</code> asettaa ne sinne.</p>
<p><strong>Pelisuunnitelma</strong> on toinen johdettu merkintä vaiheen rinnalla, ja se vastaa kysymykseen, jota nippu tallennettuja suodattimia ei osaa esittää: ”näytä virheeni holding gamessa”. Tunnus <code>gt:</code>, toistettavissa (<code>gt:holding gt:mutualholding</code>), vuorossa olevan <strong>pelaajan</strong> näkökulmasta — sen suunnitelman, jossa päätös tehtiin.</p>
<p>Kymmenen tunnistettua suunnitelmaa, siinä järjestyksessä kuin säännöt ne käyvät läpi, tarkimmasta yleisimpään:</p>
<ul>
<li><code>race</code> — molempien osapuolten takimmaiset nappulat ovat ohittaneet toisensa: kontakti ei ole enää mahdollinen. GNU Backgammonin raja.</li>
<li><code>bearin</code> — vuorossa oleva kotiuttaa nappuloitaan, kun vastustaja pitää yhä ankkuria hänen kotialueellaan.</li>
<li><code>crunch</code> — vuorossa olevalla on enintään kuusi nappulaa pisteidensä 1 ja 2 ulkopuolella. GNU Backgammonin sääntö, sen tekijän kynnysarvo.</li>
<li><code>backgame</code> — kaksi tai useampi ankkuri vastustajan kotialueella.</li>
<li><code>acepoint</code> — yksi ainoa ankkuri, vastustajan ykköspisteellä, vähintään kaksikymmentä pipiä jäljessä.</li>
<li><code>blitz</code> — kolme tai useampi kotipiste tehtynä, ja vastustaja palkilla tai yksinäinen nappula lyötävänä siellä.</li>
<li><code>primevprime</code> — molemmat pitävät vähintään neljän pisteen muuria, ja kummallakin on nappula loukussa toisen muurin takana.</li>
<li><code>mutualholding</code> — molemmat pitävät korkeaa ankkuria.</li>
<li><code>holding</code> — vuorossa oleva pitää korkeaa ankkuria, vastustaja ei.</li>
<li><code>contact</code> — kontakti, eikä mikään yllä olevista suunnitelmista. Avaus päätyy tänne.</li>
</ul>
<p>Kolme näistä säännöistä on GNU Backgammonin omia ja lähteistettyjä; loput ovat <strong>blunderDB:n sopimuksia</strong>. Backgammon-kirjallisuus kuvaa pelisuunnitelmat panematta niiden rajoille lukuja, eikä tälle ongelmalle ole julkaistu luokittelijoiden välistä yksimielisyysmittausta. Lähteistämättömät kynnysarvot — kolme kotipistettä blitzille, neljä pistettä muurille, kaksikymmentä pipiä jäljessä ace-point-pelille — todetaan siksi tässä eikä piiloteta koodiin, ja ne on versioitu: muuta ne, aja <code>blunderdb repair</code>, ja koko tietokanta merkitään uudelleen.</p>
<div class="admonition note">
<p>Asemaa kohti säilytetään yksi merkintä, vuorossa olevan pelaajan. Johdettu merkintä ei ole koskaan muokattavissa eikä sitä viedä koskaan totuutena, ja tietokanta, jonka suunnitelmia ei ole koskaan laskettu, ei palauta <code>gt:</code>-haulle mitään — kuten ei <code>ph:</code>-haullekaan.</p>
</div>
<p>Suodatin <strong>Merkitty</strong> säilyttää asemat, jotka olet merkinnyt ottelun lähdeohjelmassa. Vain eXtreme Gammon tuottaa tämän tiedon, joka tallennetaan siirto siirrolta <code>.xg</code>-tiedostoon; blunderDB lukee sen tuonnin yhteydessä ja säilyttää sen. Merkitty tuplauspäätös tuottaa kaksi merkittyä asemaa, tuplauksen ja hyväksynnän/luovutuksen, koska blunderDB jakaa kahtia sen, minkä lähdetiedosto tallentaa yhtenä päätöksenä.</p>
<div class="admonition note">
<p>Merkintä ei ole takautuva: tietokannassa jo olevat ottelut eivät sisällä tätä tietoa, koska se on olemassa vain lähdetiedostoissa. Riittää, että tuot kyseisen <code>.xg</code>-tiedoston uudelleen — tuonti tunnistaa kaksoiskappaleen eikä lisää muuta kuin merkinnät, koskematta olemassa oleviin kommentteihin tai analyyseihin. Merkintää ei voi asettaa eikä poistaa blunderDB:stä: väliaikaista työlistaa varten käytä mieluummin kokoelmaa.</p>
</div>
<p>Suodatin <strong>Kommentti</strong> tutkii asemiin liitettyjä kommentteja kolmessa toisensa poissulkevassa tilassa. <em>sisältää tekstin</em> etsii yhtä tai useampaa sanaa kommenttien tekstistä (syöttökenttä, sanat erotettuna merkillä <code>;</code>, vähintään yhden on täsmättävä); <em>on kommentti</em> säilyttää jokaisen aseman, jossa on kommentti, sisällöstä riippumatta; <em>ei kommenttia</em> säilyttää päinvastoin kommentoimattomat asemat — hyödyllistä yhdessä virhe- tai päivämääräsuodattimen kanssa laadittaessa luetteloa siitä, mitä on vielä kommentoitava.</p>
<div class="admonition note">
<p>Ottelutiedostosta (XG, GNUbg) tuodut kommentit lasketaan kommenteiksi. Pitääksesi vain omasi lisää komentoriville merkintä <code>co:user</code> (<code>co:xg</code>, <code>co:gnubg</code>, <code>co:bgf</code> ja <code>co:unknown</code> nimeävät muut alkuperät). Sitä paitsi <em>otteluun</em> tai <em>turnaukseen</em> liitetyt kommentit eivät kuulu tähän: ne kommentoivat ottelua tai turnausta, eivät sen asemia.</p>
</div>
<p><strong>Ottelut ja turnaukset</strong> -suodatin perustuu yhteiseen valitsimeen (modaali-ikkuna) numeeristen tunnisteiden kirjoittamisen sijaan: kaksi valintaruutuluetteloa, yksi otteluille ja yksi turnauksille, kumpikin tekstillä suodatettavissa (pelaaja, päivämäärä, tapahtuma otteluille; nimi, päivämäärä, paikka turnauksille), sekä <em>Kaikki</em> / <em>Ei mitään</em> -painikkeet, jotka vaikuttavat vain parhaillaan suodatettuun osajoukkoon. Turnauksen valitseminen valitsee automaattisesti (ja harmaantaa) sen jäsenottelut ottelulistassa, mikä tekee näkyväksi sen, että turnaus vastaa otteluidensa joukkoa.</p>
<p>Hakupaneelissa on vasemmassa reunassaan kolme välilehteä: <em>Recherche</em> (suodattimet), <em>Historique</em> ja <em>Enregistrés</em>. <strong>Historique</strong>-välilehti luettelee aiemmat haut päivämäärineen ja komentoineen: napsautus valitsee haun ja näyttää siihen liittyvän aseman laudalla, kaksoisnapsautus suorittaa sen uudelleen. Kunkin merkinnän voi tallentaa suodatinkirjastoon (kirjanmerkkikuvake, antamalla suodattimelle nimen) tai poistaa. <strong>Enregistrés</strong>-välilehti sisältää <strong>suodatinkirjaston</strong>: kaksoisnapsauta tallennettua suodatinta suorittaaksesi vastaavan haun uudelleen (katso Liite: Suodattimien edistynyt käyttö). Komento <code>history</code> (alias <code>hi</code>) avaa hakupaneelin.</p>
<div class="admonition tip">
<p>Katso saatavilla olevien suodattimien luettelo kohdasta komentoluettelo.</p>
</div>
<h3>Kokoelmapaneeli</h3>
<p><strong>Kokoelmat</strong>-paneeli (<em>CTRL-B</em>) hallinnoi asemakokoelmia. Kokoelmia voi luoda, nimetä uudelleen ja poistaa. Niihin voi lisätä asemia tai poistaa niitä (<em>Del</em>-näppäin, vahvistus pyydetään). Kaksoisnapsauta kokoelmaa selataksesi sen asemia <em>VASEN</em>- ja <em>OIKEA</em>-näppäimillä. Kokoelmien ja kokoelman sisäisten asemien järjestystä voi muuttaa vetämällä ja pudottamalla. Paina <em>CTRL-B</em> tai suorita komento <code>collection</code> näyttääksesi tai piilottaaksesi paneelin.</p>
<h3>Tuonti: mitä kirjoitetaan ja mitä ei koskaan</h3>
<p>Ottelun, aseman tai toisen tietokannan tuonti lisää sen, mikä puuttuu; se ei korvaa sitä, mikä on jo olemassa.</p>
<ul>
<li><strong>Asemaa ei koskaan kahdenneta.</strong> Sen tunnistaa sen identiteetti — nappulat, kuutio, nopat, pistetilanne — ei koskaan tiedosto, josta se tulee: sama asema kahdessa eri ottelussa pysyy yhtenä ainoana rivinä.</li>
<li><strong>Yksi analyysi moottoria kohti.</strong> eXtreme Gammon, GNUbg, BGBlitz ja sisäänrakennettu evaluaattori elävät rinnakkain samassa asemassa, ja Analyysi-paneeli kertoo kunkin alkuperän. Yhden tuonti ei pyyhi toista pois.</li>
<li><strong>Tuotua analyysiä ei koskaan lasketa uudelleen.</strong> blunderDB tallettaa sen sellaisenaan, tasomerkintöineen (”3-ply”, ”XG Roller++”, ”Book”), ekviteetteineen, virheineen, todennäköisyyksineen ja heiton tuureineen. Sääntö kuuluu: ”arviointi täyttää vain aukon” — tuonnin jälkeinen automaattinen analyysi käy läpi vain ne asemat, joilla ei ole <strong>yhtään</strong> analyysiä, ja <em>Analysoi vanhentuneet asemat uudelleen</em> jättää koskematta jokaiseen asemaan, jolla on tuotu analyysi (katso Asetukset).</li>
<li><strong>Saman tiedoston uudelleentuonti ei kirjoita mitään uudelleen.</strong> Ottelu tunnistetaan jo olemassa olevaksi; vain alkuperäisessä ohjelmistossa tehdyt merkinnät lisätään, koskematta kommentteihin tai analyyseihin.</li>
<li><strong>Mitä blunderDB ei koskaan kirjoita</strong>: uudelleenlaskettua tuuria — se luetaan lähdetiedostosta tai jää tuntemattomaksi — eikä rollouttia, jonka tietoja se ei avaa <code>.xg</code>-tiedostosta eikä osaa tuottaa.</li>
</ul>
<p>Kokoelma voi olla <strong>elävä</strong>: sen sisältö ei ole enää käsin tehty lista vaan <strong>haun</strong> tulos, joka lasketaan uudelleen joka avauksella. Kokoelman otsikon ◇-painike tekee siitä elävän viimeisimmällä haulla; ◈ kertoo sen jo olevan, ja sama painike palauttaa listan. Mitään ei tuhota: sen sisältämät asemat ovat yhä tallella, kun palaat.</p>
<p>Elävä kokoelma, jonka kysely sisältää tunnuksen jota tämä versio ei enää tunne, <strong>kieltäytyy avautumasta</strong> ja sanoo sen sen sijaan että palauttaisi koko tietokannan. Se on ainoa vika, jota tallennetulla suodattimella ei saa olla: laajeta hiljaisuudessa.</p>
<h3>Ottelupaneeli</h3>
<p><strong>Ottelupaneeli</strong> (<em>CTRL-Tab</em>) luettelee tuodut ottelut. Kaksoisnapsauta ottelua (tai paina <em>ENTER</em>) navigoidaksesi sen siirroissa. Komento <code>m</code> jatkaa navigointia viimeksi katsotussa ottelussa.</p>
<p>Käyttäjä voi:</p>
<ul>
<li>selata ottelun siirtoja näppäimillä <em>VASEN</em> ja <em>OIKEA</em>,</li>
<li>vaihtaa pelistä toiseen näppäimillä <em>PageUp</em> ja <em>PageDown</em>,</li>
<li>näyttää siirtojen analyysin (pelinappulat ja kuutio) painamalla <em>CTRL-L</em>,</li>
<li>vaihtaa pelinappulasiirtojen ja kuution analyysin välillä <em>d</em>-näppäimellä,</li>
<li>nähdä todella pelatun siirron korostettuna analyysissä.</li>
</ul>
<p>Kunkin ottelun viimeksi katsottu asema tallennetaan ja palautetaan automaattisesti. Näytä tai piilota paneeli painamalla <em>CTRL-Tab</em> tai suorittamalla komento <code>match</code>.</p>
<p>Rivin <strong>⊕</strong>-painike rikastaa ottelun tiedostosta. Sen takana ei ole mitään uutta: saman ottelun tuominen uudelleen toisessa muodossa rikastaa sen jo paikallaan — kanoninen tiiviste tunnistaa, että kyseessä on sama ottelu, ja toisen tiedoston analyysit ja kommentit täydentävät ensimmäistä. Painike tuo sen, että se löytyy: kukaan ei arvaa, että tuonti on myös rikastus. Seuraava raportti kertoo kumpi näistä tapahtui — ”rikastettu: 1” eikä ”tuotu: 1”.</p>
<p>Jokaisen ottelun voi viedä Jellyfish <code>.mat</code> -transkriptioksi otteluluettelon ⬇-painikkeella tai ottelun tietolomakkeen <em>.mat</em>-painikkeella.</p>
<p>Paneelin työkalupalkin <strong>Fusionner les joueurs</strong> -painike avaa ikkunan, joka luettelee kaikki tietokannan pelaajanimet otteluidensa määrän kanssa: valitse saman pelaajan eri kirjoitusasut, valitse säilytettävä kanoninen nimi ja yhdistä sitten. Hyödyllinen pelaajakohtaisten tilastojen yhtenäistämiseen, kun sama pelaaja esiintyy useilla nimillä.</p>
<p>Kun ottelu on avattu, laudan yläpuolelle ilmestyy <strong>tietopalkki</strong>: se muistuttaa läsnä olevista pelaajista (<em>pelaaja 1</em> vastaan <em>pelaaja 2</em>) sekä ottelun taustatiedoista (tapahtuma, paikka, kierros, päivämäärä ja ottelun pituus, kun nämä tiedot ovat saatavilla). Tämä palkki näytetään myös ottelutilan ulkopuolella: kun tutkittava asema (haun, kokoelman tai suoran haun tuloksena) on peräisin yhdestä tai useammasta ottelusta, palkki ilmoittaa sen <strong>alkuperän</strong> — ensimmäisen kyseessä olevan ottelun ja tarvittaessa « +N »-merkin, joka luettelee muut osoitettaessa. Erikseen tuotu asema, johon mikään ottelu ei viittaa, ei näytä mitään.</p>
<p>Avattaessa otteluita sisältävää tietokantaa <strong>Ottelut</strong>-paneeli näytetään heti ja tarkastelu alkaa suoraan ensimmäisestä asemasta, jotta navigoinnin voi aloittaa välittömästi.</p>
<div class="admonition note">
<p>Tietokannan voi avata kirjoitustilassa vain yksi ikkuna kerrallaan. Jos avaat tietokannan, joka on jo avattu toisessa blunderDB-ikkunassa, se avautuu <strong>vain luku</strong> -tilassa: selaus, haku ja analyysi ovat edelleen mahdollisia, mutta kaikki muokkaus on poistettu käytöstä ja otsikkopalkissa lukee « [vain luku] ».</p>
</div>
<div class="admonition tip">
<p>Katso saatavilla olevat pikanäppäimet kohdasta Näppäimistöoikotiet.</p>
</div>
<h3>Turnauspaneeli</h3>
<p><strong>Turnauspaneeli</strong> (<em>CTRL-Y</em>) mahdollistaa otteluiden ryhmittelyn turnauksiin järjestelmällistä seurantaa ja tapahtumakohtaista tilastollista analyysiä varten. Turnauksia voi luoda, nimetä uudelleen ja poistaa; otteluita voi liittää niihin. Stats-paneelin tilastoja voi suodattaa turnauksen mukaan. Näytä tai piilota paneeli painamalla <em>CTRL-Y</em>.</p>
<p>Turnaukset täyttyvät itsestään tuonnin yhteydessä. XG-, GnuBG- ja BGF-tiedostot nimeävät tapahtumansa; kun uusi ottelu tuodaan, blunderDB sijoittaa sen tämännimiseen turnaukseen ja luo turnauksen, jos sitä ei vielä ole. Turnauksen päivämäärä ja paikka jäävät tyhjiksi — ne täytetään täällä. Tietokannassa jo olevaa ottelua ei koskaan siirretä: sen tiedoston tuominen uudelleen ei kumoa käsin tehtyä järjestelyä.</p>
<p>Kunkin turnauksen <strong>PR</strong>-sarake näyttää <strong>viitepelaajan</strong> PR-arvon — eli sen pelaajan, joka esiintyy turnauksen useimmissa otteluissa (tasatilanteessa se, joka teki eniten päätöksiä). PR ei siis sekoita omaa peliäsi vastustajiesi peliin: omissa turnauksissasi se kuvastaa yksin sinun suoritustasi. Viitepelaajan nimi näkyy työkaluvihjeenä, kun viet osoittimen arvon päälle.</p>
<h3>Stats-paneeli</h3>
<h4>Johdanto</h4>
<p><strong>Stats-paneeli</strong> mahdollistaa oman pelitason analysoinnin ja kehityksen seuraamisen ajan myötä tietokantaan tuotujen asemien perusteella. Se laskee ja näyttää tunnusluvut <strong>PR</strong> (<em>Performance Rating</em>) ja <strong>MWC cost</strong> (Match Winning Chance cost) kaikille asemille tai suodatetulle osajoukolle.</p>
<p>Stats-paneeli on erityisen hyödyllinen seuraaviin tarkoituksiin:</p>
<ul>
<li><strong>oman tason arviointi</strong> tasovyöhykkeisiin nähden (<em>Maailmanluokka</em>, <em>Ekspertti</em>, *Edistynyt*…) kokonais-PR:n avulla;</li>
<li><strong>oman kehityksen seuranta</strong> turnaus turnaukselta tai ottelu ottelulta Progression-välilehden kaavioiden avulla;</li>
<li><strong>omien heikkouksien tunnistaminen</strong>: Erreurs-välilehti näyttää jakauman pelinappulasiirtojen ja kuutiopäätösten välillä sekä virheiden suuruusjakauman;</li>
<li><strong>vertailla tietokannan pelaajia</strong> keskenään, yksi rivi pelaajaa kohden, Pelaajat-välilehdellä — kätevä kokonaisen kilpailun seuraamiseen;</li>
<li><strong>siirtyminen suoraan asiaankuuluviin asemiin</strong> napsauttamalla mitä tahansa tunnuslukua (drill-down).</li>
</ul>
<h4>Paneelin avaaminen</h4>
<p>Avaa Stats-paneeli näin:</p>
<ul>
<li>Paina <em>CTRL-D</em>.</li>
<li>Kirjoita komento <code>stats</code> tai <code>st</code> komentoriville.</li>
</ul>
<div class="admonition note">
<p>Paneeli päivittyy automaattisesti aina, kun suodatinta muutetaan. Se ei laske tilastoja uudelleen pelkän PR ↔ MWC -vaihdon yhteydessä: backend laskee molemmat mittarit samanaikaisesti.</p>
</div>
<h4>Suodatinpalkki</h4>
<p>Paneelin yläosassa oleva suodatinpalkki mahdollistaa laskennan rajaamisen asemien osajoukkoon.</p>
<h5>Pelaajanäkökulma</h5>
<p>Pudotusvalikko <strong>Pelaaja</strong> suodattaa tilastot analysoitavan pelaajan mukaan. blunderDB valitsee automaattisesti pelaajan, jonka nimi esiintyy tietokannassa useimmin — vaihdettavissa milloin tahansa.</p>
<div class="admonition tip">
<p>Pelaajan vaihtaminen ei aiheuta tietojen menetystä; valitse vain aiempi pelaaja uudelleen luettelosta.</p>
</div>
<h5>Saatavilla olevat suodattimet</h5>
<ul>
<li><strong>Turnaukset</strong> — rajaus yhteen tai useampaan turnaukseen. Useita turnauksia voi valita samanaikaisesti.</li>
<li><strong>Päivämäärät</strong> — aikaväli (<em>Alkaen</em> … <em>Asti</em>). Jos vain alkupäivä on asetettu, uudemmat asemat sisällytetään.</li>
<li><strong>Päätöstyyppi</strong> — Kaikki / Pelinappulasiirrot / Kuutiopäätökset.</li>
<li><strong>Ottelun pituus</strong> — rajaus tiettyihin ottelupituuksiin (1, 3, 5, 7, 9, 11, 13, 15, 21 pistettä). Useita pituuksia voi yhdistää.</li>
</ul>
<p><strong>Reset</strong>-painike tyhjentää kaikki suodattimet (paitsi automaattisesti tunnistetun pelaajan).</p>
<div class="admonition note">
<p>Suodattimet tallennetaan blunderDB:n asetuksiin (<code>config.yaml</code>) ja palautetaan seuraavalla käynnistyskerralla.</p>
</div>
<h4>PR / MWC -vaihto</h4>
<p>Paneelin yläosassa oleva <strong>PR / MWC</strong> -painike vaihtaa kaikissa välilehdissä näytettävän mittarin.</p>
<p><strong>PR (Performance Rating)</strong></p>
<blockquote>
<p>Keskimääräinen ekviteettivirhe laskettua päätöstä kohti, kerrottuna 500:lla kuten eXtreme Gammon ja GNUbg tekevät: PR 5,0 vastaa 0,010:n ekviteettitappiota päätöstä kohti eli 10:tä millipistettä (mpt). Tarkka laskentasääntö — mitkä päätökset päätyvät nimittäjään, miten pistetilanne muunnetaan — on sivun Liite: Tilastomalli — XG / gnuBG / blunderDB -yhdenmukaisuus sääntö.</p>
<p>Tasovyöhykkeet, jotka paneeli piirtää kehityskäyrän taakse, ovat <strong>blunderDB:n oma suuntaa antava viitekehys</strong>: yksikään julkaisu ei ole näiden rajojen auktoriteetti. Kunkin vyöhykkeen yläraja on poissulkeva: PR 4 on <em>Edistynyt</em>, ei <em>Ekspertti</em>.</p>
<table>
<thead>
<tr>
<th>Taso</th>
<th>PR</th>
</tr>
</thead>
<tbody>
<tr>
<td>Maailmanluokka</td>
<td>&lt; 2</td>
</tr>
<tr>
<td>Expert</td>
<td>2 – 4</td>
</tr>
<tr>
<td>Edistynyt</td>
<td>4 – 6</td>
</tr>
<tr>
<td>Keskitaso</td>
<td>6 – 9</td>
</tr>
<tr>
<td>Harrastelija</td>
<td>9 – 12</td>
</tr>
<tr>
<td>Aloittelija</td>
<td>≥ 12</td>
</tr>
</tbody>
</table>
</blockquote>
<p><strong>MWC cost (Match Winning Chance cost)</strong></p>
<blockquote>
<p>Virheiden vuoksi menetetty kumulatiivinen ottelun voittotodennäköisyys koko suodatetussa aineistossa. Laskettu blunderDB:hen sisältyvällä Kazaross-XG2-MET:llä.</p>
<div class="admonition caution">
<p>MWC cost <strong>ei sovellu</strong> <em>money-game</em> -asemiin (joissa ei ole ottelupanosta). Nämä asemat jätetään pois MWC-laskennasta. MWC-arvot riippuvat käytetystä MET:stä; ne eivät ole suoraan vertailukelpoisia eri MET:ejä käyttävien ohjelmistojen välillä.</p>
</div>
</blockquote>
<p>PR ↔ MWC -vaihto on välitön: backend ei suorita uudelleenlaskentaa.</p>
<h4>HTML-raportti</h4>
<p>Paneelin otsikon <strong>HTML-raportti</strong>-painike tuottaa <strong>itsenäisen</strong> asiakirjan: yksi tiedosto, ei ulkoista kuvaa, ei etätyylitiedostoa, ei skriptiä. Kaaviot ovat upotettua SVG:tä, piirretty samalla piirtimellä kuin lauta näytöllä, sinun paletillasi. Se aukeaa missä tahansa selaimessa, kulkee sähköpostitse ja <strong>tulostuu PDF:ksi itse selaimesta</strong> — mikä säästää PDF-generaattorin mukaan ottamiselta sellaisen tuottamiseen, joka kaikilla jo on.</p>
<p>Se sisältää nykyisen alueen tunnusluvut (asemat, ottelut, lasketut päätökset, kokonais-, siirto- ja kuutio-PR), sitten <strong>kymmenen kalleinta päätöstä</strong>, kukin kaavionsa, kustannuksensa, sen ottelun josta se tulee, ja parhaan siirron kun analyysi sen antaa.</p>
<p>Raportti kantaa Tilastot-paneelin <strong>nykyistä suodatinta</strong>. Raportti joka ei kerro aluettaan on raportti jonka luvut eivät merkitse mitään: aseta suodatin — turnaus, päivämääräväli, pelaaja — ennen kuin tuotat sen.</p>
<h4>Dashboard-välilehti</h4>
<p><strong>Dashboard</strong>-välilehti antaa yhteenvetonäkymän keskeisistä tunnusluvuista.</p>
<h5>Tasokortit</h5>
<p>Kolme korttia näyttää PR:n (tai MWC:n) seuraaville:</p>
<ul>
<li><strong>PR Yhteensä</strong> — kaikki päätökset (nappulasiirrot + kuutio);</li>
<li><strong>PR Nappula</strong> — vain pelatut siirrot;</li>
<li><strong>PR Kuutio</strong> — vain kuutiopäätökset.</li>
</ul>
<p>Kortin napsauttaminen lataa analyysipaneeliin vastaavan osajoukon asemat (drill-down).</p>
<div class="admonition note">
<p>Päätösten kokonaismäärä näytetään kunkin kortin alaosassa, kun osoitin on sen päällä.</p>
</div>
<h5>Liukuva PR viimeisten N päätöksen perusteella</h5>
<p>Rivi PR- (tai MWC-) arvoja, jotka on laskettu viimeisten <em>N</em> päätöksen perusteella (N = 5, 10, 50, 100, 250, 500, 1000), mahdollistaa viimeaikaisen kehityssuunnan mittaamisen. Harmaannetut arvot vastaavat N:ää, joka on suurempi kuin käytettävissä olevien päätösten määrä.</p>
<p>Arvon napsauttaminen lataa vastaavat viimeiset <em>N</em> asemaa.</p>
<h5>Top blunders</h5>
<p>Luettelo 10 pahimmasta virheestä (tai MWC cost), lajiteltuna suuruuden mukaan laskevasti. Rivin napsauttaminen lataa kyseisen aseman analyysipaneeliin.</p>
<h4>Progression-välilehti</h4>
<p><strong>Progression</strong>-välilehti esittää tason kehityksen ajan myötä.</p>
<p>Välilehden yläreunassa <strong>tavoite</strong>: ”PR &lt; 5 kahdessatoista viikossa”. Tavoite, määräaika ja suuntaus joka kertoo mihin ollaan menossa — ei muuta. Tavoite joka alkaisi arvostella, onnitella tai muistuttaa olisi eri toiminto, ei tämä.</p>
<p><strong>Ehdota</strong>-painike ehdottaa tavoitetta nykytasosta: sen vyöhykkeen alarajaa jossa olet, eli seuraavaan siirtymistä. ”Vähän parempaa” ehdottaminen ei ankkuroituisi mihinkään; portaan ehdottaminen sanoo jotain — keskitasolta edistyneeksi siirtyminen näkyy ja kerrotaan.</p>
<p><strong>Suuntaus</strong> on pienimmän neliösumman sovitus otteluidesi PR-lukuihin, projisoituna määräaikaan. Se kieltäytyy lausumasta alle kolmen ottelun: suoran vetäminen kahden pisteen välille olisi väite jota ei voi pitää. Ja lause sanoo sen joka kerta — <em>suuntaus ei ole ennuste</em>.</p>
<p>Tavoite tallennetaan <strong>tietokannan metatietoihin</strong>, ei asetuksiin: se koskee sitä kirjastoa, joten se seuraa tiedostoa eikä konetta. Ei skeemamuutosta: <code>metadata</code> on jo avain/arvo-taulu, jonka lukevat sekä <code>blunderdb info</code> että demoni.</p>
<h5>Turnauskohtainen viivakaavio</h5>
<p>Viivakaavio näyttää PR:n (tai MWC:n) jokaiselle turnaukselle (X-akseli: turnausten järjestys, Y-akseli: mittarin arvo). Värivyöhykkeet havainnollistavat tasorajat.</p>
<p>Kaavion pisteen napsauttaminen avaa pikavalikon, jossa on kaksi vaihtoehtoa:</p>
<ul>
<li><strong>Avaa turnaus</strong> — avaa turnauksen Turnaukset-paneelissa.</li>
<li><strong>Avaa asemat</strong> — lataa turnauksen kaikki asemat analyysipaneeliin.</li>
</ul>
<h5>Ottelukohtainen hajontakaavio</h5>
<p>Hajontakaavio esittää jokaisen ottelun (X-akseli: päivämäärä, Y-akseli: PR tai MWC). Pisteen koko on verrannollinen ottelun päätösten määrään.</p>
<p>Pisteen napsauttaminen avaa pikavalikon:</p>
<ul>
<li><strong>Avaa ottelu</strong> — avaa ottelun Ottelut-paneelissa.</li>
<li><strong>Avaa asemat</strong> — lataa ottelun kaikki asemat analyysipaneeliin.</li>
</ul>
<h4>Erreurs-välilehti</h4>
<p><strong>Erreurs</strong>-välilehti erittelee virheiden lähteet.</p>
<h5>Jakauma kuutiotoimen mukaan</h5>
<p>Pylväskaavio näyttää PR:n (tai MWC:n) jokaiselle kuutiopäätöksen tyypille: <em>NoDouble</em>, <em>DoubleTake</em>, <em>DoublePass</em>, <em>TooGood</em>. Jokainen pylväs näyttää myös päätösten määrän ja blunder-osuuden työkaluvihjeessä.</p>
<p>Pylvään napsauttaminen lataa kyseistä kuutiotoimea vastaavat asemat, <strong>vain ne, joissa on virhe</strong> (drill-down).</p>
<h5>Kuutiovirheiden suunta</h5>
<p>Yllä oleva jakauma kertoo, <em>paljonko</em> kuutiopäätökset maksavat; tämä taulukko kertoo, <em>mihin suuntaan</em> ne menevät pieleen.</p>
<p>Kuutioasemaan liittyy kaksi eri pelaajan tekemää päätöstä, jotka esitetään tässä kahtena rivinä:</p>
<ul>
<li><strong>Tarjoaminen</strong> — kuutiota hallussaan pitävä pelaaja tuplaa tai jättää tuplaamatta. Hänen virheitään ovat <strong>jääneet tuplaukset</strong> (olisi pitänyt tuplata) ja <strong>ennenaikaiset tuplaukset</strong> (ei olisi pitänyt).</li>
<li><strong>Vastaaminen</strong> — pelaaja, jolle kuutio tarjotaan, ottaa vastaan tai luovuttaa. Hänen virheitään ovat <strong>virheelliset luovutukset</strong> (oikea vastaanotto luovutettiin) ja <strong>virheelliset vastaanotot</strong> (oikea luovutus otettiin vastaan).</li>
</ul>
<p>Kaksi riviä pidetään erillään tarkoituksella: pelaaja voi aivan hyvin tuplata myöhään <em>ja</em> ottaa vastaan löysästi, ja yksi ainoa tunnusluku kutsuisi sitä ”tasapainoiseksi” ja hukkaisi tiedon molemmat puoliskot.</p>
<p>Kussakin ruudussa näkyy päätösten määrä; työkaluvihje antaa kertyneen menetetyn equityn. Ruudun napsautus lataa vastaavat asemat. Nollassa oleva ruutu ei ole napsautettava.</p>
<div class="admonition note">
<p>Tämä taulukko laskee päätöksiä, se ei tuomitse. Se, mistä erosta alkaen taipumus ansaitsee nimen, riippuu otoskoosta ja vertailukohdasta, eivätkä ne ole moottorin tietoja.</p>
</div>
<h5>Checker / Cube -vertailu</h5>
<p>Vertailukaavio asettaa rinnakkain pelinappulasiirtojen ja kuutiopäätösten PR:n. Pylvään napsauttaminen lataa osajoukon asemat, joissa on virhe.</p>
<h5>Virheiden suuruusjakauman histogrammi</h5>
<p>Histogrammi jakaa virheet niiden suuruuden mukaan millipisteinä (mpt, luokat: 0–5, 5–10, 10–25, 25–50, 50–100, ≥ 100). Pylvään napsauttaminen lataa luokan asemat.</p>
<h4>Erittelyt-välilehti</h4>
<p><strong>Erittelyt</strong>-välilehti jakaa samat päätökset, jotka kokonaisluvut laskevat, neljälle akselille. Yksikään niistä ei määrittele uudelleen, mikä on päätös: se olisi toinen PR samalla nimellä.</p>
<ul>
<li><strong>Pelin vaiheen mukaan</strong> — avaus, keskipeli, kilpajuoksu, nappuloiden poisto. Tämä vastaa kysymykseen ”PR:ni kilpajuoksussa verrattuna PR:ääni kontaktissa”. Merkintä lasketaan laudasta (katso Hakupaneeli); tietokanta, jonka vaiheita ei ole koskaan laskettu, sijoittaa kaiken kohtaan <em>Luokittelematon</em>, ja <code>blunderdb repair</code> täyttää sen.</li>
<li><strong>Pelisuunnitelman mukaan</strong> — kilpajuoksu, blitz, ankkuri, backgame, muuri muuria vastaan… Tämä on se erittely, jota varten luokittelija on olemassa: ”missä häviän eniten?”, suunnitelma suunnitelmalta. Sama johdettu merkintä kuin vaiheella, samat varaukset, ja <code>blunderdb repair</code> täyttää sen samoin.</li>
<li><strong>Merkinnän mukaan</strong> — kommentteihin kirjoitetut <code>#sana</code>. Asemalla voi olla useita: <strong>nämä rivit eivät summaudu kokonaismäärään</strong>, ja paneeli sanoo sen taulukon alla. Merkintä nimeää, se ei jaa osiin.</li>
<li><strong>Tilanteen mukaan</strong> — molempien puolten puuttuvat pisteet, luettuna vuorossa olevan puolelta, siis päättäjän puolelta. <em>Money</em>-rivi on raha peli. Alle kymmenen päätöksen solu on <strong>harmaannettu, määrä yhä näkyvissä</strong>, eikä piilotettu: liian vähän luettavaksi, mutta puute pysyy tarkistettavana.</li>
</ul>
<div class="admonition note">
<p>Crawford-peliä ei eroteta: blunderDB ei tallenna tuota tietoa asemaan. Käytännön vaikutus on pieni — Crawford-pelissä ei ole lainkaan tuplauspäätöstä — mutta puute on todellinen, ja se on parempi kirjoittaa kuin jättää arvattavaksi.</p>
</div>
<h4>Harjoittelu ja oikea peli</h4>
<p>Komento <code>blunderdb list --type study --days 30</code> asettaa kolme lukua rinnakkain, pelisuunnitelma kerrallaan: kuinka monta <strong>eri asemaa</strong> jaksolla kerrattiin, mikä PR oli <strong>ennen</strong> sitä, mikä PR on <strong>sen jälkeen</strong>.</p>
<p>Kolme lukua, eikä neljättä. <strong>Ei hyötysaraketta eikä nuolta</strong>, koska mikään tässä ei vakioi mitään: pelaaja on voinut kohdata vahvempia vastustajia, vaihtaa formaattia tai yksinkertaisesti pelata enemmän kilpajuoksuja tänä kuussa. Yhteys on lukijan tekemä; efektiä julistava sarake väittäisi syy-yhteyttä, jota nämä tiedot eivät kanna. Luvut sen sijaan ovat tarkkoja.</p>
<p>Kertaukset lasketaan <strong>eri asemina</strong>: neljä kertaa kuussa kerrattu kortti on yksi opiskeltu asema, ja toistojen laskeminen saisi kuukauden pänttäämisen näyttämään kuukauden kattavuudelta. PR:n päätökset sen sijaan lasketaan kaikki — jokainen tehtiin kerran. Alle kymmeneen päätökseen nojaava PR näkyy merkkinä <code>—</code>, otoskoko näkyvissä vieressä.</p>
<h4>Pelaajat-välilehti</h4>
<p>Neljä edellistä välilehteä kuvaavat <strong>yhtä</strong> pelaajaa; <strong>Pelaajat</strong>-välilehti vertaa kaikkia. Se näyttää yhden rivin kutakin tietokannan pelaajaa kohti, mikä vastaa koko kilpailua seuraavan järjestäjän tarpeeseen yksittäisen pelaajan sijaan.</p>
<p>Sarakkeet järjestyksessä:</p>
<table>
<thead>
<tr>
<th>Sarake</th>
<th>Merkitys</th>
</tr>
</thead>
<tbody>
<tr>
<td>Pelaaja</td>
<td>Nimi <strong>sellaisena kuin se otteluissa esiintyy</strong>. Kahdella kirjoitusasulla tallentunut pelaaja näkyy siis kahdella rivillä; käytä pelaajien yhdistämistä niiden liittämiseen yhteen.</td>
</tr>
<tr>
<td>Ottelut</td>
<td>Valitulla ajanjaksolla pelattujen otteluiden määrä.</td>
</tr>
<tr>
<td>V–T</td>
<td>Voitot ja tappiot. Kesken jäänyt ottelu (katkennut loki, luovutus) ei laske kumpaakaan: V + T voi siis olla pienempi kuin otteluiden määrä.</td>
</tr>
<tr>
<td>Päätökset</td>
<td>Laskettujen päätösten määrä — PR:n nimittäjä. Tämä sarake kertoo, mitä viereiset luvut ovat arvoltaan: kahdestatoista päätöksestä laskettu PR ei merkitse mitään.</td>
</tr>
<tr>
<td>PR</td>
<td>Kokonais-Performance Rating.</td>
</tr>
<tr>
<td>Nappula-PR, kuutio-PR</td>
<td>PR eriteltynä päätöstyypin mukaan.</td>
</tr>
<tr>
<td>Snowie</td>
<td>Snowie Error Rate (katso Liite: Tilastomalli — XG / gnuBG / blunderDB -yhdenmukaisuus).</td>
</tr>
<tr>
<td>Karkeat virheet</td>
<td>Vakavien virheiden määrä (vähintään 0,100 EMG).</td>
</tr>
<tr>
<td>Tuuri</td>
<td>Keskimääräinen tuuri heittoa kohden, millipisteinä (mpt), etumerkillä: positiivinen, jos nopat olivat suotuisat.</td>
</tr>
</tbody>
</table>
<p>Käyttö:</p>
<ul>
<li><strong>Järjestä</strong> — napsauta sarakeotsikkoa. Taulukko avautuu nousevan PR:n mukaan, paras pelaaja ensin. Pelaajat, joista ei ole mitattu mitään, pysyvät alimpina järjestyssuunnasta riippumatta: tiedon puutteesta johtuva nolla ei ole täydellinen suoritus.</li>
<li><strong>Avaa pelaajan tiedot</strong> — napsauta riviä. Pelaaja valitaan suodatinpalkissa ja näkymä vaihtuu Dashboard-välilehdelle.</li>
<li><strong>Rajaa ajanjaksoa</strong> — päivämäärä-, turnaus- ja ottelupituussuodattimet toimivat tavalliseen tapaan, joten taulukon voi rajata yhden kilpailun päiviin.</li>
</ul>
<div class="admonition note">
<p>Tällä välilehdellä <strong>Pelaaja</strong>-luettelo ja <strong>päätöstyypin</strong> valinta ovat poissa käytöstä: taulukko näyttää kaikki pelaajat ja erittelee nappula- ja kuutiopäätökset jo omiin sarakkeisiinsa.</p>
</div>
<div class="admonition important">
<p>Viiva (”—”) merkitsee arvoa, jota <strong>ei ole koskaan mitattu</strong>; sitä ei pidä sekoittaa nollaan. Näin on erityisesti Tuuri-sarakkeen kohdalla kaikissa otteluissa, jotka tuotiin ennen skeemaversiota 2.15.0: tuuria ei silloin tallennettu, eikä mikään salli sen palauttamista jälkikäteen — lähdetiedostot on tuotava uudelleen. Muodot, jotka eivät sitä kuljeta (BGF, Jellyfish <code>.mat</code>), eivät sitä koskaan tarjoa.</p>
</div>
<h4>Yhdistämissääntö</h4>
<div class="admonition important">
<p>Turnauksen (tai minkä tahansa osajoukon) PR lasketaan <strong>summa/summa</strong>-säännöllä — ei koskaan yksittäisten otteluiden PR:ien keskiarvona.</p>
<p>Kaava:</p>
<pre class="math">PR_&#123;turnaus&#125; = 500 \\times \\frac&#123;\\sum_&#123;i&#125; \\text&#123;virhe&#125;_i&#125;&#123;\\text&#123;päätösten kokonaismäärä&#125;&#125;</pre>
<p><strong>Esimerkki:</strong> pelaaja pelaa kaksi ottelua turnauksessa —</p>
<ul>
<li>Ottelu A: 10 päätöstä, 0,100 ekviteettiä menetetty → PR = 5,0</li>
<li>Ottelu B: 90 päätöstä, 0,540 ekviteettiä menetetty → PR = 3,0</li>
</ul>
<p>Naiivi PR:ien keskiarvo: (5,0 + 3,0) / 2 = <strong>4,0</strong> <em>(virheellinen)</em></p>
<p>Summa/summa-sääntö: 500 × 0,640 / (10 + 90) = <strong>3,2</strong> <em>(oikein)</em></p>
<p>Summa/summa-sääntö on ainoa, joka käsittelee oikein otteluiden vaihtelevat pituudet (21 pisteen ottelu painaa enemmän kuin 1 pisteen ottelu).</p>
</div>
<h4>MWC: rajoitukset</h4>
<ul>
<li>MWC cost lasketaan <strong>Kazaross-XG2-MET</strong>:stä, joka on kilpailullisen backgammonin de facto -viitetaulukko. Tulokset eivät ole suoraan verrattavissa ohjelmistoihin, jotka käyttävät muita METejä. Se on sama taulukko, luettuna saman sisääntulokohdan kautta, jota sisäänrakennettu evaluaattori käyttää pistetilanteen mukaisiin kuutiopäätöksiinsä: tilastot ja moottori eivät voi poiketa toisistaan tässä. Se antaa omat arvonsa 25 tehtävään pisteeseen asti kummallakin puolella; sen ylitse sitä jatketaan Zadehin taulukolla, joka lasketaan kuten GNUbg:ssä, 64 pisteeseen asti.</li>
<li><em>Money-game</em> -asemat (joissa ei ole ottelun pistetilannetta) <strong>jätetään pois</strong> MWC-laskennasta. Jos tietokantasi sisältää paljon money-game-asemia, MWC cost voi olla aliarvioitu tai ei käytettävissä.</li>
<li>MWC cost on kumulatiivinen koko suodatetussa aineistossa — ei päätöskohtainen tunnusluku. Se mittaa virheidesi kokonaisvaikutusta voittomahdollisuuksiisi.</li>
</ul>
<h3>Eval-paneeli</h3>
<p><strong>Eval</strong>-paneeli (<em>CTRL-E</em>) arvioi reaaliajassa laudalle asetetun aseman, oli se mikä tahansa; bearoff-asemassa se erikoistuu ja laskee lisäksi EPC:n (Effective Pip Count). Se avataan painamalla <em>CTRL-E</em>, napsauttamalla alapaneelin Eval-välilehteä tai suorittamalla komento <code>epc</code>. Tämä komento säilyttää alkuperäisen nimensä: paneelin nimi oli ensin <em>EPC</em>, sitten <em>Bearoff</em>, ennen kuin siitä tuli <em>Eval</em> — täältä on siis etsittävä sitä, mitä aiempi versio kutsui Bearoff-paneeliksi, sillä tuo nimi tarkoittaa enää bearoff-taulukoiden asetusvälilehteä.</p>
<p>Paneeli näyttää aina sen <strong>ainoan päätöksen</strong>, jota laudalle asetettu asema vaatii — ei koskaan kahta yhtä aikaa — sekä siihen liittyvät faktat. Kukin suure luetaan sille sopivalla akselilla yhden pakotetun akselin sijaan: kummankin pelaajan voitto-, gammon- ja backgammon-todennäköisyys sekä cubeless-ekviteetti, jotka lasketaan <em>ennen heittoa</em>, luetaan <strong>pelaajakohtaisesti</strong> (ala, ylä, sitten Δ) kuutiopäätöksen vasemmalla puolella, kun noppia ei ole asetettu. Faktat ja päätös pysyvät vierekkäin: kuutiopäätös ei koskaan siirry sitä perustelevien lukujen alapuolelle, olipa käyttöliittymän kieli tai laudan asema mikä tahansa. Heti kun noppia asetetaan, nämä samat <em>ennen heittoa</em> -arvot vaihtavat akselia: ne luetaan <strong>vuorossa olevan pelaajan kannalta</strong> ehdokassiirtojen luettelon kärjessä kursivoituna <em>ennen heittoa</em> -rivinä — ei yhtenä ehdokassiirtona lisää, vaan kiintopisteenä, jota vasten kukin siirto luetaan. Tämän rivin ja siirron välinen ero sisältää heiton tuurin, ei koskaan siirron ansiota, eikä sillä siksi ole virhesaraketta. Puhtaassa bearoff-asemassa toinen taulukko, aina <strong>pelaajakohtainen</strong> ja aina läsnä, nopat asetettuina tai ei, sisältää EPC:n, pip countin, wastagen, keskimääräisen heittomäärän ja keskihajonnan; nämä viisi saraketta eivät koskaan siirry. Kaksi taulukkoa on pinottu ja ne jakavat saman sarakeruudukon: samat reunat, samat sarakemerkit, yksi ainoa pistesarake — ne luetaan yhtenä kaksikerroksisena kokonaisuutena. Tilamerkki, moottorin attribuutio (myös viimeisimmän evaluoinnin syvyys näkyy siinä) ja <em>Haaste</em>-valintaruutu muodostavat erillisen kaistan, joka on tasattu oikealle taulukoiden yläpuolelle.</p>
<p>Vain ehdokassiirtojen luettelo vierii — myös <em>ennen heittoa</em> -rivi pysyy kiinnitettynä sen yläpuolella; muu paneeli (faktat, merkki, kuutiopäätös) pysyy aina näkyvissä ilman erityistä paneelin koon säätöä.</p>
<p>Faktataulukon ja päätöksen laskee sisäänrakennettu gammonNet ilman XG:tä tai gnubg:tä. Laskenta seuraa asemaa jäädyttämättä koskaan käyttöliittymää: 0-ply-syvyys näytetään heti jokaisen eleen jälkeen, ja puolen sekunnin paikallaanolon jälkeen syvempi evaluointi (oletuksena 2 plyä, säädettävissä asetusten <em>gammonNet</em>-välilehdellä) korvaa sen taustalla — mikä tahansa uusi ele peruuttaa tämän taustalaskennan. Merkkikaistalla tai kilpajuoksuasemassa tilamerkin sisällä näytetty syvyys on aina se, joka todella tuotti näytetyn luvun, ei koskaan pyydetty syvyys; sitä ei toisteta joka rivillä, koska reaaliaikainen evaluointi käyttää samaa syvyyttä kaikille siirroille. Ehdokassiirtojen ja kuutiopäätöksen ekviteetti noudattaa aseman pistetilannetta: money gamessa se ilmaistaan pisteinä, ottelun pistetilanteessa <strong>normalisoituna ekviteettinä</strong> — samalla asteikolla kuin XG:ssä ja GNU Backgammonissa, jossa nykyisen kuution arvon voittaminen on +1 ja sen häviäminen −1 — eikä niitä koskaan sekoiteta samassa taulukossa. Sarakkeen otsikko ilmoittaa sen selvästi sen sijaan, että asteikko jätettäisiin arvattavaksi: ”Equity (money)” money gamessa, ”Equity (match)” ottelun pistetilanteessa. Se ottaa huomioon <strong>elävän kuution</strong>: haku arvostaa jokaista loppuasemaa kuutiomallilla (Janowski, mitattu tehokkuus) aseman kuutiotilassa, samaan tapaan kuin XG ja GNU Backgammon tekevät <em>cubeful</em>-arvioinnissa. Tämä tekee gammon-go- ja gammon-save-vaikutukset näkyviksi pistetilanteessa — pistetilanteessa 4-away/2-away jäljessä oleva pelaaja pelaa 8/2 6/2 avaussiirrolla 6-4, koska varhainen tuplaus antaa gammonille ottelun arvon, mitä kuutioton arviointi ei näe. <em>Ennen heittoa</em> -rivi puolestaan pysyy <strong>cubeless</strong>-ekviteettinä: se on aseman fakta, ei päätös. Tämä paneeli ei koskaan muuta tietokantaa: kyseessä on laskenta, ei tallennettu analyysi. Ehdokassiirron napsauttaminen näyttää sen laudalla nuolina, täsmälleen kuten Analyysipaneelissa. Huomaamaton <strong>?</strong>-painike merkkikaistalla vie moottorin <code>gammonNet &lt;https://github.com/kevung/gammonNet&gt;</code>_ -tietovarastoon; täydellinen attribuutio (Strehlin verkko, gammonNet-kokoonpano) on ohjeen Kiitokset-osiossa.</p>
<p>Käyttäjä muokkaa nappuloiden asemaa koko laudalla täsmälleen kuten muokkaustilassa: vasen napsautus asettaa alapelaajan nappulan, oikea napsautus yläpelaajan nappulan. Toinen, kilpajuoksun taulukko ilmestyy vain, kun saatu asema on puhdas bearoff (molempien pelaajien kaikki nappulat kotikentässään); missä tahansa muussa asemassa vain neljän yhteisen sarakkeen taulukko (voitto, gammon, backgammon, cubeless) reagoi, ja päätös koskee nappuloita tai yleistä kuutiota sen mukaan, onko noppia asetettu.</p>
<p>Kussakin faktataulukossa on yksi rivi pelaajaa kohden — tunnistettavissa värillisestä pisteestä, musta pelaaja aina alimpana. Ensimmäinen sisältää, niin kauan kuin noppia ei ole asetettu, pelaajan voiton, gammonin ja backgammonin (todennäköisyydet ilman %-merkkiä) sekä cubeless-ekviteetin; toinen, bearoff-asemassa nopat asetettuina tai ei, EPC:n, pip countin, wastagen (EPC:n ja pip countin erotus), keskimääräisen heittomäärän ja keskihajonnan. Kun molemmilla pelaajilla on vertailtavia arvoja, <strong>Δ</strong>-rivi antaa <em>etumerkilliset</em> erotukset (ala − ylä: negatiivinen, kun musta pelaaja on edellä). Muussa kuin kilpajuoksuasemassa noppien asettaminen kadottaa siis itse faktataulukot: niiden neljä saraketta vaihtoivat juuri akselia vuorossa olevan pelaajan kannalle siirtoluettelon kärkeen.</p>
<p>Kuutiopäätöksellä on aina sama muoto lukujen alkuperästä riippumatta — tarkka taulukko, evaluoitu tila tai tavallinen gammonNet-evaluointi: <strong>yksi rivi vaihtoehtoa kohden</strong>, järjestyksessä <em>ei tuplausta</em>, <em>tuplaus/ota</em>, <em>tuplaus/ohita</em>, kullakin ekviteettinsä aseman viitekehyksessä ja erotuksensa parhaaseen vaihtoehtoon. Järjestys ei koskaan muutu, toisin kuin siirtoluettelossa: kolmella vaihtoehdolla on nimi, joten luetaan nimeä, ei sijaa. Parhaan tunnistaa korostuksesta ja tyhjäksi jätetystä erotussolusta. Kun kuutio on jo käännetty, vaihtoehdot ovat <em>ei uudelleentuplausta</em>, <em>uudelleentuplaus/ota</em>, <em>uudelleentuplaus/ohita</em>.</p>
<p>Viimeinen rivi antaa <strong>päätöksen</strong>. Sillä on neljä arvoa: <em>ei tuplausta</em>, <em>tuplaus, ota</em>, <em>tuplaus, ohita</em> ja <em>liian hyvä tuplattavaksi</em>, viimeksi mainittu silloin, kun aseman pelaaminen tuottaa enemmän kuin pisteen lunastaminen: tuplaaminen olisi tällöin virhe päinvastaisesta syystä kuin tavallisessa <em>ei tuplausta</em> -tapauksessa. Tämä on myös ainoa paikka, jossa paneeli sanoo, ettei päätöstä <strong>ole</strong>, sen sijaan että antaisi ymmärtää laskennan olevan kesken:</p>
<ul>
<li><em>ei päätöstä</em> — tila ei oikeuta siihen; kuutiopäätöstä ei koskaan arvioida (katso <em>arvioitu</em>-merkki);</li>
<li><em>ei evaluoitavissa tässä pistetilanteessa</em> — moottori hylkää aseman, tyypillisesti pistetilanteen, joka on match-ekviteettitaulukon horisontin ulkopuolella, toisin sanoen jommallakummalla puolella on yli 64 pistettä tehtävänä;</li>
<li><em>vastustajan kuutio</em> ja <em>kuollut kuutio (Crawford)</em> — kuutiota ei voi kääntää. Ekviteetit näytetään edelleen tiedoksi, mutta millään vaihtoehdolla ei ole erotusta: virhe on se, mitä valinta maksaa, eikä valintaa ole.</li>
</ul>
<p>Money gamessa asemassa voimassa olevat <strong>Jacoby</strong>- ja <strong>Beaver</strong>-säännöt näkyvät kuutiotaulukon alla, pieninä merkkeinä sen päätöksen vieressä, jota ne muuttavat: aseman ”ei tuplausta” -päätös Jacoby-säännön alaisena ei ole sama laskelma kuin ilman sitä, eikä mikään muu näytöllä kertonut sitä.</p>
<p>Kolmas merkki, <strong>Kuution katto</strong>, ilmestyy, kun lähdetunniste rajaa kuution — sekä ottelutilanteessa että money gamessa. Se ei kuvaa yllä näkyvää laskentaa: sisäänrakennettu arviointi ei mallinna kattoa, joten tuomio koskee vapaata kuutiota. Juuri siksi merkki on siinä: katolla rajattu kuutio on ainoa näkyvä syy, jonka takia blunderDB ja eXtreme Gammon voivat ilmoittaa samasta asemasta kaksi eri tuomiota.</p>
<p>Tilamerkki, evaluointisyvyys, linkki moottoriin ja <em>Haaste</em>-valintaruutu muodostavat erillisen kaistan, joka on tasattu oikealle taulukoiden yläpuolelle.</p>
<p><strong>Vuorossa olevaa pelaajaa</strong> ja <strong>kuution sijaintia</strong> muokataan suoraan laudalla kuten muokkaustilassa: pelaajan bearoff/pistetilanne-suorakulmion napsauttaminen antaa vuoron hänelle; kuution napsauttaminen kierrättää tilat keskellä → alapelaajan hallussa → yläpelaajan hallussa (oikea painike vastakkaiseen suuntaan). Kuution arvo pysyy kiinnitettynä — money gamessa ekviteetit ilmaistaan nykyisen kuution yksiköissä, vain sen omistajalla on merkitystä. Analyysi lasketaan heti uudelleen. Arvioidussa tilassa itse merkki on napsautettavissa ja avaa suoraan asetusten <em>Bearoff</em>-välilehden; sen työkaluvihje selittää miksi (kuutiopäätöstä ei voi arvioida, <code>ADR-0009 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0009-race-win-chances-are-read-or-convolved-cube-verdicts-are-never-estimated.md&gt;</code>__) ja miten tarkkaa aluetta laajennetaan.</p>
<p>Myös <strong>pistetilannetta</strong> muokataan suoraan laudalla kuten muokkaustilassa: vasen napsautus pelaajan pistetilannesuorakulmioon vähentää hänen tarvitsemiensa pisteiden määrää, oikea napsautus lisää sitä. Kun <em>money</em>-pistetilanteesta (-1, -1) poistutaan muokkaamalla vain toista puolta, toinen puoli tasataan automaattisesti samaan arvoon epäjohdonmukaisen pistetilanteen sijaan. Bearoff-asemassa <em>tarkassa</em> tilassa siirtyminen money-pistetilanteesta ottelun pistetilanteeseen jättää voittotodennäköisyyden ennalleen (tietokannasta luettu arvo, joka pätee viitekehyksestä riippumatta), mutta vaihtaa näytetyn ekviteetin ja kuutiopäätöksen <em>evaluoidun</em> tilan arvoihin — tarkka taulukko on rakenteeltaan money-taulukko, eikä se osaa vastata pistetilanteessa esitettyyn kysymykseen. Merkistä tulee tällöin yhdistelmä (« tarkka (voitto) · evaluoitu (kuutio) »), jotta tämä sanotaan suoraan.</p>
<p>Myös <strong>nopat</strong> muokataan samalla tavalla, ja juuri ne ratkaisevat, mikä kysymys esitetään: asetetut nopat tekevät nappulapäätöksen (ehdokassiirtojen listan), noppien puuttuminen kuutiopäätöksen. Vasen napsautus nopalla kasvattaa sen arvoa (6 palaa 1:een), oikea napsautus pienentää sitä (1 palaa 6:een); nopan napsauttaminen laudalla, jolla ei ole noppia, asettaa kaksi kerralla — yksi noppa ei olisi nappulapäätös eikä kuutiopäätös. Pelaajan suorakulmion napsauttaminen poistaa nopat kuutiokysymyksen esittämiseksi, ja seuraava napsautus nopalla palauttaa ne ennalleen.</p>
<p><em>ASKELPALAUTIN</em> tai kaksoisnapsautus laudan ulkopuolella tyhjentää aseman: tyhjä lauta, money-pistetilanne (-1, -1), ei asetettuja noppia — Eval-paneelin omat arvot, jotka eroavat muokkaustilan arvoista (7 molemmilla, nopat 3-1), jotta ne pysyvät johdonmukaisina paneelin oletusnäytön kanssa.</p>
<h4>Tuplauskuution matriisi</h4>
<p>Kuutiopäätös ei ole laudan ominaisuus. Samat nappulat ja sama pip-luku tuplataan tilanteessa 2-away/4-away eikä tuplata tilanteessa 4-away/2-away; se joka on oppinut money-vastauksen on oppinut yhden ruudun ruudukosta. Eval-paneeli näyttää sen ruudun, jonka asema kantaa; <strong>tuplauskuution matriisi</strong> näyttää koko ruudukon.</p>
<p>Komento <code>cm</code> avaa sen näytöllä olevalle asemalle. Kukin ruutu antaa tuomion yhdessä pistetilanteessa: rivi on vuorossa olevan pelaajan vielä tarvitsemien pisteiden määrä, sarake vastustajan. Neljä tuomiota kirjoitetaan <em>ET</em> (ei tuplausta), <em>TO</em> (tuplaus, otto), <em>TP</em> (tuplaus, pass) ja <em>LH</em> (liian hyvä); moottorin hylkäämässä ruudussa on kysymysmerkki, ja osoitin kertoo syyn sekä ruudun kolme ekviteettiä. Tarjolla on kolme ottelupituutta: 5, 7 ja 9 pistettä.</p>
<p>Aseman pistetilanne korvataan kunkin ruudun tilanteella; sen <strong>kuutio</strong> säilyy. Ruudukko vastaa kysymykseen, missä pistetilanteessa kääntäisin <em>tämän</em> kuution, ei siihen mitä keskitetty asema tekisi. Se on kauttaaltaan Crawfordin jälkeinen: Crawford-pelissä kuutio ei ole pelissä, eikä sarake ”et saa tuplata” kertoisi asemasta mitään.</p>
<p>Jokainen ruutu on oma hakunsa. Moottori ottaa pistetilanteen huomioon — se ei pelaa samaa peliä tilanteessa 2-away kuin 7-away — joten yksi ainoa haku luettuna eri otteluekviteettien läpi olisi väärässä juuri siellä, missä pistetilanne merkitsee. Ruudukko saapuu ensin 0-plyllä ja laskeutuu uudelleen määritetyllä näyttösyvyydellä, kun ikkuna on levossa: sama porrastus kuin muualla paneelissa, ja 9 pisteen ruudukko maksaa noin puolitoista sekuntia.</p>
<p>Sama ruudukko lasketaan käyttöliittymän ulkopuolella komentorivin komennolla cubematrix.</p>
<h4>Aseman tuominen Eval-paneeliin</h4>
<p>Paneeli avautuu oletuksena bearoff-asemaan, mutta tutkiminen alkaa useimmiten jo käsillä olevasta asemasta. Kaksi elettä tuo sen paneeliin:</p>
<ul>
<li><strong>Oikea napsautus laudalla</strong> analyysipaneelissa tai ottelua selattaessa ja sitten <em>Evaluoi tämä asema</em>: Eval-paneeli avautuu suoraan tähän asemaan sellaisena kuin se näytetään. Pikavalikko ei ilmesty Eval-paneelissa eikä Hakupaneelissa, joissa oikea painike on jo varattu toisen värin nappuloiden asettamiseen.</li>
<li><strong>CTRL-C ja sitten CTRL-V</strong>: kopioi asema analyysipaneelista ja liitä se sitten Eval-paneelissa. Liittäminen hyväksyy myös muualta tulevan tunnisteen — XGID:n (eXtreme Gammon, GNU Backgammon, toinen blunderDB-instanssi) tai OGID:n (OpenGammon): riittää, että se on leikepöydällä.</li>
<li><strong>Komento</strong> <code>import XGID=…</code> (tai <code>import OGID=…</code>) siihen tapaukseen, ettei tunniste ole leikepöydällä vaan viestissä, päätteessä luetulla foorumilla tai skriptin tuottamana. Se on sama verbi kuin pelkkä <code>import</code>: ilman argumenttia se avaa tiedostovalitsimen, argumentin kanssa se lukee tunnisteen. Polku on sen jälkeen sama kuin liittämisessä — sama luku, sama kaksoiskappaleiden poisto, sama tuodun aseman avaus.</li>
</ul>
<p>OGID kantaa vain aseman: ei arviota eikä kommenttia. Asema saapuu siis ilman analyysiä, aivan kuten paljas XGID, ja sisäänrakennettu arviointi voi täyttää aukon jälkikäteen.</p>
<p>Eval-paneelin lauta on luonnos: asema saapuu sinne ilman tietokantatunnistettaan, joten mikään täällä tehty muutos ei voi kirjoittaa uudelleen tietuetta, josta se on peräisin. Kaikki tavalliset laudan muokkaukset ovat siinä käytettävissä (nappulat, kuutio, nopat, pistetilanne), ja evaluointi seuraa jokaista muutosta.</p>
<p>Toiseen suuntaan <em>CTRL-C</em> kopioi Eval-paneelin laudan leikepöydälle asetetuista nappuloista uudelleen lasketun XGID:n kera — joten sen voi liittää suoraan eXtreme Gammoniin tai toiseen blunderDB-instanssiin. Vain asema matkustaa: paneelin näyttämä evaluointi ei ole tietokannan tietue eikä seuraa kopion mukana.</p>
<p>Eval-paneelista poistuttaessa aiemmin tarkasteltu asema palautetaan: luonnosta ei koskaan tallenneta itsestään.</p>
<p>Kun asema on puhdas bearoff (molempien pelaajien kaikki nappulat kotikentässään) eikä noppia ole asetettu, kuutiopäätös näyttää vuorossa olevalle pelaajalle:</p>
<ul>
<li><em>tarkassa</em> tilassa: money-ekviteetit (cubeless, ei tuplausta, tuplaus/ota, tuplaus/ohita) ja <strong>money-kuutiopäätöksen</strong> (ei tuplausta, tuplaus/ota, tuplaus/ohita tai liian hyvä tuplattavaksi) — ottelun pistetilanteen ulkopuolella, katso yllä pistetilanteen tapaus,</li>
<li><em>evaluoidussa</em> tilassa: samat ekviteetit ja sama neljän arvon päätös, mutta <strong>gammonNetin pelaamina</strong> (haku + Janowskin kuutiomalli) taulukosta lukemisen sijaan — käytettävissä <strong>myös ottelun pistetilanteessa</strong>, mitä arvioitu tila ei ole koskaan voinut tarjota;</li>
<li><em>arvioidussa</em> tilassa: kuutiopäätöstä ei tällöin tarkoituksella näytetä — vain voittotodennäköisyys faktataulukossa virhemarginaaleineen jää käytettäväksi.</li>
</ul>
<p>Heti kun kilpajuoksuasemaan asetetaan noppia, tämä <em>ennen heittoa</em> -kuutiopäätös katoaa — lauta pyytää tällöin nappulapäätöstä, ei kuutiopäätöstä — mutta voittotodennäköisyys pysyy aseman faktana, ei päätöksenä: se siirtyy <em>ennen heittoa</em> -riville siirtoluettelon kärkeen EPC:n viereen, joka puolestaan pysyy näkyvissä aivan vasemmalla.</p>
<p>Merkki ilmaisee tilan: <strong>tarkka</strong> (two-sided-tietokannasta luettu arvo), <strong>evaluoitu · &lt;syvyys&gt;</strong> (gammonNetin pelaama — näytetty syvyys on se, joka todella tuotti näytetyn luvun), <strong>arvioitu ± marginaali</strong> tai, ottelun pistetilanteessa tarkalla alueella, <strong>tarkka (voitto) · evaluoitu (kuutio)</strong> — katso yllä. Tarkka tila voittaa kaikkialla, missä se on käytettävissä; muuten evaluoitu tila näytetään heti, kun laskenta on valmis, ja se korvaa paikallaan odotuksen aikana näytetyn arvioidun tilan. Katso Eval-paneelin metodologia ja oletukset kolmen tilan ja niiden oletusten täsmällinen määritelmä.</p>
<p><strong>Tarkan alueen laajentaminen.</strong> Ensimmäisellä käynnistyksellä laskettu taulukko kattaa 6 nappulaa puolellaan. Kaksi tapaa mennä pidemmälle, asetusten <em>Bearoff</em>-välilehdellä:</p>
<ul>
<li>laskea laajempi kaksipuolinen taulukko — TS-06-15:een asti, jos koneella on siihen muistia. Välilehti kertoo koon, muistin ja ajan tällä koneella ennen aloitusta, ja laskenta menee tauolle ja jatkuu. Peruttu laskenta jättää <code>.part</code>-tiedoston, jota ei koskaan lueta taulukkona;</li>
<li>osoita mikä tahansa gnubg:n two-sided <code>.bd</code>-tiedosto. Alueeltaan laajin tietokanta voittaa automaattisesti.</li>
</ul>
<p><strong>Paneelin lauta on luonnos, ja se muistetaan.</strong> Eval-paneelista poistuminen ja sinne palaaminen löytää aseman, johon se jätettiin, ei oletusarvoista ulosmenolautaa: se tarjotaan vain, kun paneeli avataan istunnossa ensimmäisen kerran. Tietokannasta paneeliin lähetetty asema voittaa tämän muistin, ja <em>ASKELPALAUTIN</em> palauttaa oletuslaudan milloin tahansa. Matkan varrella ei kirjoiteta mitään tietokantaan — luonnoksella ei ole aseman identiteettiä, ja sen arvio lasketaan perillä uudelleen sen sijaan että se kuljetettaisiin mukana.</p>
<p><strong>Haastetila.</strong> Merkkikaistan <em>Haaste</em>-valintaruutu ottaa käyttöön harjoittelutilan: jokaisen aseman muutoksen jälkeen kolmen alueen arvot piilotetaan (korvataan merkinnällä « ··· »); alueen napsautus paljastaa vain kyseisen alueen. Ilman noppia alueet ovat alapelaajan rivi, yläpelaajan rivi ja kuutiopäätös — Δ-rivi ilmestyy vasta, kun molemmat pelaajarivit on paljastettu. Päätöslohko säilyttää tällöin kolme riviään: sen arvot, päätös ja parhaan vaihtoehdon korostus katoavat, muuten harjoitus ratkeaisi etsimällä lihavoitu rivi. Kun kilpajuoksuasemaan on asetettu noppia, kummankin pelaajan EPC-rivi piilotetaan kuten ennenkin, mutta kolmas alue kattaa tällöin <em>ennen heittoa</em> -rivin ja siirtoluettelon <strong>yhdessä</strong>: koska luettelo on järjestetty parhaasta siirrosta huonoimpaan, sen osittainen paljastaminen antaisi jo vastauksen. Kun noppia on asetettu muuhun kuin kilpajuoksuasemaan, tämä sama yksi alue kattaa yksinään kaiken, mitä paneeli näyttää. Näin voi harjoitella kummankin puolen EPC:n arviointia ja sitten kuutio- tai siirtopäätöksen tekemistä ennen tarkistusta. Asetus muistetaan.</p>
<p>Sulje Eval-paneeli painamalla <em>CTRL-E</em> tai vaihtamalla toiseen välilehteen.</p>
<h4>Eval-paneelin metodologia ja oletukset</h4>
<p>Jokainen paneelin näyttämä arvo perustuu täsmällisiin oletuksiin, jotka esitetään tässä tyhjentävästi.</p>
<p><strong>Alue.</strong> <em>Kilpa-alue</em> — voittotodennäköisyys ja kuution tuomio — kattaa vain puhtaat bearoffit: molempien pelaajien kaikki jäljellä olevat nappulat omalla kotialueellaan. Asema arvioidaan <em>ennen heittoa</em>; asetetut nopat jätetään huomiotta.</p>
<p><strong>EPC-lohkot</strong> sen sijaan yltävät pidemmälle: puoli saa EPC:nsä heti kun sen kaukaisin nappula mahtuu ladattuun yksipuoliseen taulukkoon. Oletustaulukolla (kuusi pistettä) tämä on vanha kotialuesääntö; kahdeksan pisteen taulukolla, joka lasketaan <em>Bearoff</em>-välilehdeltä, puolta jonka nappula on 8-pisteellä kohdellaan kuten muitakin. Mitään ei ekstrapoloida: yhden pisteen liian kaukana oleva nappula ei yksinkertaisesti saa EPC:tä, aivan kuten 7-pisteen nappula ei saanut sitä ennen. Kun vastannut taulukko ei ole kuuden pisteen taulukko, sen nimi näkyy kilpalohkon kulmassa (”OS-08”) — ilman sitä luettaisiin oletuksena ”kuusi” ja uskottaisiin puolen olevan kokonaan kotona.</p>
<p><strong>EPC-lohkot (aina tarkkoja).</strong> EPC, keskimääräinen heittojen määrä ja keskihajonta tulevat tarkasta jakaumasta heittojen määrälle, joka tarvitaan kaikkien nappuloiden ulos saamiseen, luettuna GNUbg:n yksipuolisesta tietokannasta (6–10 pistettä, 15 nappulaa, koneella laskettu). EPC = keskimääräiset heitot × 49/6 (49/6 ≈ 8,167 on tarkka pippien keskiarvo heittoa kohden, tuplat laskettuna neljästi); wastage = EPC − pippiluku. Ainoa idealisointi on <em>yksipuolinen optimaalinen peli</em>: kumpikin pelaaja minimoi omat heittonsa vastustajasta välittämättä — se on EPC:n vakiomääritelmä.</p>
<p><strong>Voittotodennäköisyys, tarkka tila.</strong> Suora luku laajimmasta käytettävissä olevasta two-sided-tietokannasta (ensimmäisellä käynnistyksellä laskettu TS-06-06, ulkoinen tiedosto tai <em>Bearoff</em>-välilehdeltä laskettu TS-06-11). Nämä tietokannat ovat tulosta täydellisestä taaksepäin etenevästä analyysistä molempien puolten optimaalisella two-sided-pelillä: ei lisäoletuksia, virhe rajoittuu kvantisointiin (&lt; 0,002 %).</p>
<p><strong>Voittotodennäköisyys, arvioitu tila.</strong> Tietokannan alueen ulkopuolella: todennäköisyys saadaan konvoloimalla kaksi one-sided-jakaumaa (vuorossa oleva pelaaja voittaa, jos hänen heittomääränsä on pienempi tai yhtä suuri kuin vastustajan) ja soveltamalla sitten kiinteää polynomikorjausta, joka on kalibroitu etukäteen TS-06-11-tietokantaa vasten. Kolme oletusta:</p>
<ul>
<li>kahden ulosottoprosessin <strong>riippumattomuus</strong> — rakenteellinen ominaisuus kilpajuoksussa: ilman kontaktia ei ole mitään vuorovaikutusta;</li>
<li><strong>molempien puolten optimaalinen one-sided-peli</strong> — tämä on <em>approksimaatio</em>: todellisuudessa jäljessä oleva pelaaja poikkeaa siitä pelatakseen varianssia ja johdossa oleva pelatakseen varman päälle. Mitattu vaikutus on antisymmetrinen harha (konvoluutio liioittelee johtajan etumatkaa), jonka korjaus absorboi tilastollisesti;</li>
<li><strong>korjaus</strong> on kalibroitu ja validoitu oraakkelin alueella (enintään 11 nappulaa pelaajaa kohden). Mitattu jäännösvirhe: keskihajonta 0,05 %, 99. persentiili 0,17 %, suurin havaittu 0,9 % (voittotodennäköisyyden prosenttiyksikköinä). <strong>Kun nappuloita on pelaajaa kohden yli 11, tämä raja on ekstrapoloitu</strong> — suuntaus on monotoninen, mutta mikään oraakkeli ei todenna sitä.</li>
</ul>
<p><strong>Ekviteetit ja kuutiopäätös (vain tarkka tila).</strong> Näytetyt ekviteetit ovat <strong>money gamen, ilman Jacobya</strong>, bearoff-kirjallisuuden viitekehyksessä. Alueella ≤ 11 nappulaa pelaajaa kohden gammonit ovat mahdottomia (kumpikin puoli on jo ottanut ulos vähintään 4 nappulaa): tämä ei ole approksimaatio. Päätös (ei tuplausta / tuplaus, ota / tuplaus, ohita) rekonstruoidaan tarkasti tallennetuista ekviteeteista GNUbg:n säännön mukaan, joka on validoitu kohta kohdalta sen analyysiä vasten.</p>
<div class="admonition note">
<p>Cubeful-ekviteetit olettavat <strong>molempien puolten optimaalisen kuutiopelin loppuun asti</strong>: tulevat uudelleentuplaukset arvostetaan täysimääräisesti (täydellinen taaksepäin etenevä analyysi). Pelin lopun hyvin epävakaissa kilpajuoksuissa uudelleentuplausten ketju syö lähes koko vuorossa olevan puolen edun — ekviteetit « ei tuplausta » ja « tuplaus/ota » voivat tällöin olla lähellä nollaa siinä, missä XG:n kaltainen moottori, jonka kuutiomalli ei arvosta tätä ketjua, näyttää dead cubea lähellä olevia arvoja (esimerkiksi 2 nappulaa pisteessä 3 vastaan 2 nappulaa pisteessä 2: 62 %:n voittotodennäköisyys, tarkka D/T +0,006, XG:llä +0,475). Näytetty <strong>päätös</strong> sitä vastoin yhtyy moottoreiden päätökseen.</p>
</div>
<p><strong>Voittotodennäköisyys ja päätös, evaluoitu tila.</strong> Tarkan alueen ulkopuolella voittotodennäköisyys tulee gammonNetin raakatulosteesta (0- tai 2-plyn haku eleen mukaan, ei koskaan taulukosta luettuna) ja päätös tähän tulosteeseen sovelletusta Janowskin « Decide »-mallista — haku <em>pelaa</em> trajektorin sen sijaan, että tiivistäisi siitä hetkellisen tilannekuvan, mikä on täsmälleen se, mitä arvioitu tila ei voinut tehdä (katso alempana), ja mahdollistaa, ainoana kolmesta tilasta tarkan ohella, päätöksen <strong>ottelun pistetilanteessa</strong>.</p>
<p>Tämä tila on mitattu, ei vain oletettu, sisäänrakennettua two-sided-taulukkoa vasten (<code>TestEvalMeasure</code>, 4000 satunnaisotannalla poimittua money-päätöstä, kanoniset parametrit 2 plyä k=12): money-päätösten yhtäpitävyys <strong>93,4 %</strong> (3735/4000), eriteltynä etäisyyden mukaan gammonNetin ottopisteestä — 61,1 % alle 1 %:n päässä ottopisteestä (kolikonheitolle herkin alue), 88,3 % välillä 1–5 %, 91,5 % välillä 5–10 %, 94,0 % välillä 10–20 %, 94,4 % sen yli. Voittotodennäköisyyden ero: keskiarvo 0,85 %, mediaani 0,44 %, 95. persentiili 3,21 %, maksimi 8,30 %. Cubeful-ekviteetin ero: keskiarvo 0,039, mediaani 0,018, 95. persentiili 0,151, maksimi 0,406. Muoto on odotettu: valtaosa erimielisyydestä keskittyy täsmälleen ottopisteeseen, jossa kaksi perustellusti erilaista menetelmää eroavat eniten tiukassa päätöksessä — ei hajanainen virhe, joka maksaisi ekviteettiä kaikkialla.</p>
<p>Tämä mittaus koskee <strong>money</strong>-päätöksiä kilpajuoksussa. Ottelun pistetilanteen mukaisesta tuomiosta — jonka vain tämä tila osaa antaa — ja kontaktiasemista ei ole julkaistua mittausta: edellä sanottu ei siirry näihin tapauksiin.</p>
<p><strong>Miksi ei syvemmälle kuin 2 plytä?</strong> Koska mittaus sanoo, ettei se tuota mitään. Nappulapäätös maksaa samalla koneella 99 ms 2 plyllä ja 8,4 s 3 plyllä — <strong>kahdeksankymmentäviisi kertaa enemmän</strong>. Neljästäkymmenestä molemmilla syvyyksillä toistetusta oikeasta päätöksestä syvempi haku muutti mielensä <strong>kahdesti</strong>, ja molemmilla kerroilla hyöty, jonka se itselleen luki, oli korkeintaan 0,0005 normalisoitua ekviteettiä: kaksi kertaluokkaa alle 0,020:n, sen kynnyksen, jonka jälkeen eXtreme Gammon ylipäätään puhuu virheestä. Päätöstä kohti, kaikki tapaukset yhdessä, hyöty on 0,0000.</p>
<p>Asetusta ei siis tarjota. Tämä ei sano, että 3 plytä olisi yleisesti arvoton, vaan että <em>tällä</em> verkolla, kanonisella suodattimella, se ei maksa paneelin ääressä istuvan odotusta. Mittaus on toistettavissa (<code>TestThreePlyMeasure</code>), ja johtopäätös ratkaistaan uudelleen, jos verkko muuttuu.</p>
<p><strong>Miksi arvioitua päätöstä ei ole?</strong> Seuraava koskee nimenomaan <em>konvoluutio</em>-menetelmää (arvioitu tila), ei yllä kuvattua evaluoitua tilaa: cubeful-ekviteetti on <em>trajektoriongelma</em> (milloin tuplata), jota mikään aseman tilastollinen tiivistelmä ei tavoita — paras mitattu staattinen malli jättää jäännösvirheen (ekviteetin keskihajonta 0,016, maksimi 0,20), joka riittää kääntämään kaikki tiukat päätökset. Samoin päätöksen muuntaminen ottelun pistetilanteeseen match-ekviteettitaulukon avulla mitattiin riittämättömäksi (12 % erimielisyyksiä GNUbg:n 2-ply-analyysin kanssa, mukana todellisia blundereita). Koska itsevarmasti näytetty väärä päätös on pahempi kuin ei päätöstä lainkaan, konvoluutiolla ei ole koskaan ollut oikeutta näyttää päätöstä — tämän aukon täyttää haku, joka pelaa trajektorin, ei tilastollinen tiivistelmä.</p>
<div class="admonition note">
<p>Bearoff-tietokannat ovat muuttumattomia matemaattisia taulukoita. blunderDB laskee ne itse, samoin kuin GNUbg:n <code>makebearoff</code>-työkalu — tavu tavulta — asetusten <em>Bearoff</em>-välilehdellä tai komennolla <code>blunderdb bearoff generate</code>.</p>
</div>
<h3>Anki-paneeli</h3>
<p><strong>Anki-paneeli</strong> (<em>CTRL-K</em>) mahdollistaa asemien opiskelun välitoistolla FSRS-algoritmia käyttäen. Käyttäjä voi luoda pakkoja kokoelmista tai hakutuloksista.</p>
<p><strong>Pakkojen luominen:</strong> Napsauta <em>New Deck</em> luodaksesi pakan kokoelmasta tai nykyisistä hakutuloksista. Hakuun perustuvat pakat synkronoituvat automaattisesti, kun Anki-välilehti avataan.</p>
<p><strong>Kertaaminen:</strong> Valitse pakka ja napsauta <em>Study</em> (tai kaksoisnapsauta pakkaa) aloittaaksesi erääntyneiden korttien kertaamisen. Jokainen kortti näyttää vastaavan aseman laudalla. Arvioi muistamisesi näppäimillä <em>1</em> (Uudelleen), <em>2</em> (Vaikea), <em>3</em> (Hyvä) tai <em>4</em> (Helppo). Paina <em>Esc</em> lopettaaksesi ja palataksesi pakkaluetteloon.</p>
<p><strong>Kuutiopäätöksistä tulee kaksi korttia, ketjutettuina.</strong> Kuutiopäätös on kaksi kysymystä — ”tuplaus?”, sitten ”hyväksy?” — ja blunderDB on aina tallentanut ne kahtena asemana. Pakka, joka valitsee vain toisen puolikkaan, saa toisenkin: päätös täydennetään, ei laajenneta. Ja kun molemmat ovat vuorossa, toinen tulee <strong>heti</strong> ensimmäisen jälkeen.</p>
<p>Kumpikin säilyttää oman arvosanansa ja oman aikataulunsa: nämä eivät ole yhden kortin kaksi vaihetta, vaan kaksi korttia. Ketjutus ei aikaista mitään eräpäivää — se järjestää jo erääntyneet kortit, ei muuta. Koska ne syntyvät yhdessä, ne erääntyvät yhdessä ensimmäisellä kerralla, ja juuri siinä siitä on hyötyä.</p>
<p><strong>Vastauksen näyttäminen:</strong> Kortti esittää kysymyksen — mikä siirto pelataan tai mikä kuutiotoimi tehdään. Mieti, ja paina sitten <em>VÄLILYÖNTI</em> (tai napsauta peitettyä aluetta) paljastaaksesi vastauksen: aseman tallennetun analyysin sellaisena kuin Analyysi-välilehti sen esittää. Se ilmestyy arviointipainikkeiden alle, jotka pysyvät paikoillaan ja ulottuvilla. Listan siirron napsauttaminen näyttää sen laudalla.</p>
<p>Mikään ei pakota paljastamaan vastausta arviointia varten: jos olet varma, näppäimet *1*–*4* pysyvät käytössä. Vastaus peittyy uudelleen seuraavan kortin kohdalla, mutta ei silloin, kun vain vaihdat välilehteä — käy katsomassa Eval-paneelia tai aseman kommenttia, vastaus odottaa palatessasi.</p>
<p>Asema, jolla ei ole tallennettua analyysiä, ilmoittaa sen suoraan ilman peitettyä aluetta.</p>
<p><strong>Istunnon rajaaminen.</strong> Oletuksena kertausistunto käy läpi kaikki erääntyneet kortit. Voit rajata sen korttimäärään pakkakohtaisesti Asetuksissa: rastita <em>Rajaa istunto</em> ja ilmoita, montako korttia istunnon tulee tarjota. Kun raja täyttyy, istunto päättyy ja kertoo siitä — viesti erottaa tilanteen ”raja täynnä, näin monta korttia yhä erääntyneenä” aidosti tyhjästä jonosta. Jos haluat silti jatkaa, vapaa harjoittelu on olemassa: se tarjoaa muita asemia muuttamatta aikataulusta mitään.</p>
<p>Raja <strong>0</strong> ei tarjoa yhtään korttia: se on oma tilansa, hyödyllinen pakan jäädyttämiseen turnaukseen valmistautumisen ajaksi, eikä se ole sama asia kuin ”ei rajaa”. <em>Study</em>-painike on tällöin pois käytöstä.</p>
<p>Raja koskee <strong>istuntoa</strong>, ei päivää. blunderDB-pakka rakentuu kokoelman tai haun varaan: se on äärellinen aineisto, joka esitellään muutamassa istunnossa ja jonka päivittäistä määrää sen koko jo rajaa. Päiväkatto ei puraisi koskaan, tai sitten se loisi ruuhkan pakalle, joka mahtui yhteen istuntoon.</p>
<p><strong>Vapaa harjoittelu (cram):</strong> <em>Cram</em>-painike <em>Study</em>-painikkeen vieressä aloittaa vapaan harjoittelusession: sinulle näytetään satunnaisia asemia pakasta FSRS-aikataulusta riippumatta. Tämä tila <strong>ei koskaan muuta kertausaikataulua</strong> — ihanteellinen lämmittelyyn ennen turnausta tai teemapakan tehokkaaseen kertaamiseen ilman, että sen järjestys häiriintyy. <em>Cram</em>-merkki korvaa kortin tilan, ja <em>Seuraava</em>-painike (näppäimet *1*–*4*) selaa asemia. <em>Esc</em> palaa luetteloon tallentamatta keskeytettyä sessiota.</p>
<p><strong>Kortin siirtäminen syrjään ilman arvosanaa.</strong> Kertauksen aikana kortin otsikon oikea napsautus tarjoaa kolme elettä, jotka poistavat sen istunnosta kertomatta ajoittajalle mitään:</p>
<ul>
<li><strong>Keskeytä</strong> — kortti säilyttää aikataulunsa eikä tule enää esiin niin kauan kuin se on keskeytettynä. Näin siirretään syrjään väärä tai vielä hyödytön kortti menettämättä siihen liittyvää historiaa.</li>
<li><strong>Hautaa</strong> — kortti katoaa seuraavaan päivään asti. Toisin kuin keskeytys, tämä ei sano mitään sen arvosta: se on sille, jonka on juuri nähnyt muualla tai jota ei halua kohdata kahdesti samana iltana.</li>
<li><strong>Poista</strong> — kortti lähtee pakasta vahvistuksen jälkeen. Asema itse jää tietokantaan: pakka on opiskelulista kirjaston yli, ei koskaan sen kopio.</li>
</ul>
<p>Mikään näistä kolmesta ei kirjaa arvosanaa: syrjään siirretty kortti ei ole vastattu kortti, eikä se lasketa istunnon summaan.</p>
<p><strong>Kertausloki.</strong> Pakan asetuksissa <em>Kertausloki</em>-painike näyttää, mitä ajoittajalle <strong>kerrottiin</strong> — päivä, asema, arvosana, tila, myönnetty väli — vastakohtana sille, mitä se suunnittelee. Vain täällä näkee vahingossa annetun arvosanan. Siellä sitä ei voi korjata: aikataulu pysyy ulottumattomissa, ja juuri se sääntö tekee lokista hyödyllisen — menneisyyttä ei voi kirjoittaa uusiksi, mutta sen voi tietää.</p>
<p><strong>Keskeytys/Jatkaminen:</strong> Voit keskeyttää kertausistunnon milloin tahansa näppäimellä <em>Esc</em>. Painike muuttuu muotoon <em>Resume</em> ja näyttää edistymisesi. Napsauta sitä jatkaaksesi siitä, mihin jäit.</p>
<p><strong>Pakkojen hallinta:</strong> Toimintopainikkeilla voi nimetä uudelleen, synkronoida, nollata tai poistaa pakkoja (kahdesta viimeksi mainitusta pyydetään vahvistus). FSRS-parametrit (tavoitepysyvyys, enimmäisväli, satunnaisuus) asetetaan pakkakohtaisesti Asetuksissa (rattikuvake).</p>
<p><strong>Pysyvyys: tavoite ja mittaus.</strong> <em>Tavoitepysyvyys</em> on sinun valintasi työmäärän ja mieleenpalautuksen laadun välisessä vaihtokaupassa: mitä korkeampi se on, sitä lyhyemmiksi välit käyvät ja sitä enemmän kertaat. Sen rinnalla Asetukset näyttävät <strong>mitatun pysyvyyden</strong> omista kertauksistasi — tieto, ei koskaan ohjaus: blunderDB ei muuta tavoitettasi jahdatakseen onnistumisprosenttiasi. Alle parinkymmenen kertauksen mittausta ei näytetä: se luettaisiin tosiasiaksi, vaikka se on pelkkää kohinaa.</p>
<p>Pysyvyyden muuttaminen <strong>ei vaikuta taannehtivasti</strong>: kukin kortti omaksuu uuden tahdin seuraavassa kertauksessaan, eivätkä jo asetetut eräpäivät siirry. Vaikutus on siis vähittäinen eikä näy samana päivänä.</p>
<p><em>Enimmäisväli</em> rajaa välistyksen. Äskettäin luotu pakka lähtee vuodesta: asema, jonka algoritmi siirtäisi useiden vuosien päähän, on poistunut pakasta ilman että olet niin päättänyt, ja oma pelisi muuttuu sitä nopeammin. Vanhemmat pakat säilyttävät sen arvon, joka niillä oli.</p>
<h3>Mikroharjoitukset</h3>
<p>Anki-paneeli kertaa <strong>arviota</strong>; mikroharjoitukset harjoittavat kolmea <strong>laskutoimitusta</strong>, jotka tehdään pelipöydässä kellon käydessä ja joita mikään kertausväli ei kasvata. Komento <code>train</code> aloittaa viiden kysymyksen istunnon:</p>
<ul>
<li><code>train pips</code> — laske vuorossa olevan pelaajan pipit näytetystä asemasta.</li>
<li><code>train epc</code> — arvioi saman pelaajan EPC kilpajuoksuasemasta, jonka moottori osaa arvioida.</li>
<li><code>train tp</code> — palauta mieleen pitkän kilpajuoksun hyväksymispiste satunnaisesti arvotussa tilanteessa, taulukon <code>tp2_live</code> mukaan.</li>
</ul>
<p>Kysymys ON näytetty asema: lauta on sovelluksen oma, ja sen yläpuolinen palkki kantaa vain kysymyksen, syötteen ja korjauksen. Vastaus kirjoitetaan ja vahvistetaan näppäimistöllä (<em>Enter</em> tarkistaa ja siirtyy eteenpäin, <em>Esc</em> poistuu istunnosta).</p>
<p>Toleranssi riippuu harjoituksesta, ja se sanotaan eikä arvata: pip-laskennassa sitä <strong>ei ole</strong> — yhden pipin päähän oikea yhteenlasku on väärä yhteenlasku — EPC sallii puoli pipiä, hyväksymispiste kaksi prosenttiyksikköä. Lopuksi istunto näyttää oikeiden vastausten määrän ja <strong>mediaaniajan</strong> kysymystä kohti.</p>
<p>Vain tämä yhteenveto säilytetään, tietokannan metatiedoissa: istunto ei säilytä jälkeä kysymys kysymykseltä, eikä mitään kirjoiteta ennen kuin se on päättynyt. Kesken poistuminen ei siis tallenna mitään.</p>
<h4>Tietovisa: harjoittelun PR</h4>
<p><code>train quiz</code> esittää neljännen lajin kysymyksiä. Anki-paneeli panee ulkoa opettelemaan; tietovisa <strong>testaa</strong>. Selatusta listasta arvotaan viisi jo analysoitua asemaa, ja päätös on tehtävä:</p>
<ul>
<li>siirtopäätöksessä kirjoita siirto näppäimistöllä notaatiossa (<code>13/7 8/7</code>);</li>
<li>kuutiopäätöksessä napsauta <em>Ei tuplausta</em>, <em>Tuplaus, hyväksy</em> tai <em>Tuplaus, luovuta</em>.</li>
</ul>
<p>Analyysipaneeli pysyy peitettynä, kunnes kysymykseen on vastattu: se kantaa vastauksen, eikä kysymys, jonka vastaus näkyy vieressä, ole kysymys.</p>
<p>Korjaus pitää kolme lopputulosta erillään, ja niiden sekoittaminen valehtelisi. <strong>Sääntöjenvastainen siirto</strong> ei ole huonosti valittu siirto — se on sääntövirhe. <strong>Laillinen siirto, jota moottori ei arvioinut</strong>, ei ole virhe lainkaan: sillä ei yksinkertaisesti ole hintaa, eikä se maksa istunnolle mitään. Arvioitu siirto maksaa sen, minkä analyysi sanoo, millipisteinä.</p>
<p>Lopuksi istunto näyttää <strong>tietovisan PR:n</strong>, joka lasketaan samalla kaavalla kuin tilastot laskevat oikealle pelille — 500 × keskimääräinen virhe normalisoituna ekviteettinä. Juuri se tekee luvuista vertailukelpoisia: tietovisan PR 6 ja ottelun PR 6 mittaavat samaa asiaa samalla asteikolla.</p>
<h3>Metatietopaneeli</h3>
<p><strong>Metatietopaneeli</strong> näyttää nykyisen tietokannan yleiset tiedot: nimi, kuvaus, asemien määrä, otteluiden ja pelien määrä, skeeman versio. Käytettävissä komennolla <code>meta</code>.</p>
<p>Se näyttää myös tietokannan alkuperän, <strong>jos sellainen on</strong> — ks. Tietokannan jakaminen: alkuperä ja salasana. Tavallisessa tietokannassa tätä osiota ei näy.</p>
<h3>Tietokannan jakaminen: alkuperä ja salasana</h3>
<p>Asemakokoelmaa jakavalla opettajalla on käytössään kaksi toisistaan riippumatonta mekanismia, molemmat vapaaehtoisia ja <strong>vientihetkellä</strong> valittavia: tiedoston merkitseminen alkuperällään ja sen suojaaminen salasanalla.</p>
<div class="admonition note">
<p>Kumpikaan ei seuraa, mitä tiedostolle tapahtuu. blunderDB <strong>ei tallenna mitään tietokannan vastaanottajan puolella</strong>: merkityn tietokannan avaaminen on täsmälleen samanlaista kuin minkä tahansa muun avaaminen, eikä missään kirjata, kuka sen avasi, milloin, tai mistä sen sisältö on peräisin.</p>
</div>
<h4>Tietokannan merkitseminen alkuperällään</h4>
<p>Vienti-ikkuna mahtuu yhdelle näytölle: lomake ja sen päälle kirjoituksen ajaksi asettuva edistymisnäkymä. Ikkuna sulkeutuu itsestään valmistuttuaan, ja tulos näkyy tilapalkissa.</p>
<p>Kolme seikkaa ansaitsee huomiota:</p>
<ul>
<li><strong>Vienti koskee parhaillaan näkyvissä olevia asemia</strong>, ei koko tietokantaa. Haun jälkeen viedään vain tulokset — ikkuna muistuttaa siitä yläreunassa.</li>
<li><strong>Kokoelma, jonka asemat eivät kaikki sisälly valintaan, saapuu vaillinaisena.</strong> Siksi luettelo näyttää kunkin kokoelman katetun osuuden (”12/40”) ja merkitsee sen punaisella, kun se on osittainen.</li>
<li><strong>Turnaukset voi viedä vain otteluiden kanssa</strong>: ilman niitä turnaus–ottelu-yhteyttä ei ole ja turnaus saapuisi tyhjänä. Valintaruutu pysyy poissa käytöstä, kunnes ”sisällytä ottelut” on valittu.</li>
</ul>
<p>Kentät <em>Käyttäjä</em>, <em>Kuvaus</em> ja <em>Päivämäärä</em> kuvaavat <strong>syntyvää tiedostoa</strong>; ne on esitäytetty lähdetietokannasta. Valintaruutu <em>Omat tallennetut suodattimet</em> on erillinen muista: se ei vie sisältöä vaan omat tallennetut hakusi, joista ei ole hyötyä jonkun toisen tietokannassa.</p>
<p>Valinta <strong>Merkitse tämä tiedosto alkuperällään</strong> tuo näkyviin kaksi kenttää:</p>
<ul>
<li><strong>Alkuperä</strong> — mikä tämä tiedosto on ja mistä se tulee, omin sanoin: ”Jean Dupontin oppitunti — 12. maaliskuuta 2026”. Tämä kenttä on <strong>pakollinen</strong>: niin kauan kuin se on tyhjä, vientipainike pysyy passiivisena.</li>
<li><strong>Huomautus</strong>, valinnainen — käyttöehdot, yhteysosoite, pyyntö olla jakamatta eteenpäin.</li>
</ul>
<p>Merkintä allekirjoitetaan merkitsijän identiteetilläsi. Se on siten <strong>muuttumaton ja väärentämätön</strong>: kukaan ei voi muuttaa sitä eikä valmistaa sellaista sinun nimissäsi. Se ei sen sijaan ole <strong>poistamaton</strong> — jaettu tiedosto on tavallinen SQLite-tietokanta ja blunderDB on vapaa ohjelmisto. Merkintä ei estä mitään: se kertoo, mistä tiedosto on peräisin.</p>
<h4>Merkitsijän identiteetti</h4>
<p>Merkinnät allekirjoitetaan <strong>merkitsijän identiteetilläsi</strong>, joka syntyy itsestään ensimmäisellä kerralla, kun merkitset tiedoston; mitään ei tarvitse määrittää. Se kuuluu henkilölle eikä tietokannalle: kaikissa tiedostoissasi on sama julkinen sormenjälki muotoa <code>A3F1-9C24-7B05-E1D8</code>.</p>
<p>Voit kertoa tämän sormenjäljen vastaanottajillesi, jotta he voivat varmistaa tiedoston tulevan todella sinulta. Identiteetti siirtyy koneelta toiselle yhtenä tiedostona (pääte <code>.bdbid</code>), haluttaessa salalauseella suojattuna. <strong>Tällä tiedostolla voi allekirjoittaa sinun nimissäsi: älä jaa sitä.</strong></p>
<p>Asetuksissa (työkalurivin rataskuvake) <em>Merkitsijän identiteetti</em> -välilehti näyttää nimesi ja sormenjälkesi ja tarjoaa toiminnot <em>Tallenna identiteetti…</em>, <em>Lataa identiteetti…</em> ja <em>Luo uusi…</em>.</p>
<div class="admonition warning">
<p><strong>Uuden luominen ei mitätöi mitään.</strong> Merkintä sisältää sen allekirjoittaneen julkisen avaimen: se todentuu siis ikuisesti, aivan itsekseen. Jos identiteettitiedostosi on vuotanut, sen haltija voi jatkaa allekirjoittamista vanhalla sormenjäljelläsi, ja nuo merkinnät pysyvät pätevinä.</p>
<p>Vuodon jälkeen sinua ei suojaa ohjelmisto: suoja on siinä, että julkaiset uuden sormenjälkesi ja ilmoitat vastaanottajillesi vanhan pätemättömäksi.</p>
<p>Uuden luominen korvaa nykyisen avaimen; blunderDB tarjoaa mahdollisuutta tallentaa se ennen korvaamista.</p>
</div>
<h4>Tietokannan suojaaminen salasanalla</h4>
<p>Salasana kirjoitetaan peitettynä, sekä tässä että suojattua tiedostoa avattaessa; silmäkuvake näyttää sen <strong>niin kauan kuin sitä pidetään painettuna</strong> ja peittää sen taas heti, kun ote irrotetaan.</p>
<p>Valinta <strong>Suojaa tämä tiedosto salasanalla</strong> tuottaa tiedoston, jonka pääte on <code>.dbx</code> — myös silloin, kun olit valinnut tallennusikkunassa <code>.db</code>-päätteisen nimen, sillä tuo ikkuna avautuu ennen salasanan kysymistä. Avaamiseen käytetään tavallista tietokannan avausta: valintaikkuna hyväksyy sekä <code>.db</code>- että <code>.dbx</code>-tiedostot. blunderDB kysyy tällöin salasanan ja asentaa viereen tavallisen tietokannan; sen jälkeen mitään ei enää kysytä.</p>
<p>Ikkuna tarjoaa mahdollisuutta <strong>poistaa suojattu tiedosto avaamisen jälkeen</strong>: muuten sama sisältö jää talteen kahdella nimellä. Ruutu ei ole oletuksena valittuna — suojattu tiedosto jää sinulle, jos aiot välittää sen eteenpäin — ja poisto tapahtuu vasta onnistuneen avaamisen jälkeen.</p>
<div class="admonition warning">
<p>Salasana suojaa tiedoston <strong>siirron ajaksi</strong>, ei itse tietokantaa. Se estää sivullista avaamasta latauskansioon unohtunutta tiedostoa tai vahingossa edelleen lähetettyä liitettä. Se ei suojaa siltä, jolle olet antanut salasanan.</p>
</div>
<p>Salasana tarkistetaan <strong>joka</strong> avauskerralla, myös silloin kun tiedosto on jo aiemmin avattu tällä koneella.</p>
<p>Teknisesti tietokanta salataan <strong>AES-256:lla GCM-tilassa</strong>, ja avain johdetaan salasanasta <strong>Argon2id</strong>-funktiolla (64 MiB muistia, 3 kierrosta, 4 säiettä) käyttäen satunnaista, tiedostokohtaista suolaa. GCM-tila todentaa kokonaisuuden: väärä salasana havaitaan sellaiseksi, samoin mikä tahansa salatun tiedoston muuttaminen — vioittunutta tietokantaa ei koskaan saada hiljaisesti.</p>
<p>Suojatun tiedoston otsake pysyy <strong>salaamattomana</strong>: sen alkuperä on luettavissa ilman salasanaa.</p>
<h4>Tiedoston alkuperän lukeminen</h4>
<p>Avaa sovelluksessa tiedosto ja näytä <strong>Metatiedot</strong>-paneeli (komento <code>meta</code>). Paneelin yläosaan ilmestyy vain luettava <strong>Alkuperä</strong>-osio, joka kertoo, mitä on kirjattu, kenen toimesta, milloin, ja missä tilassa allekirjoitus on:</p>
<ul>
<li>”✓ allekirjoitus varmennettu — sinun merkitsemäsi”: tiedostossa on oma merkintäsi ehjänä;</li>
<li>”✓ allekirjoitus varmennettu”: merkintä on ehjä ja peräisin toisesta avaimesta — vertaa sen sormenjälkeä siihen, jonka tekijä on sinulle kertonut;</li>
<li>”⚠ virheellinen allekirjoitus”: asiakirjaa on muutettu tai se on väärennetty.</li>
</ul>
<p>Tavallisessa tietokannassa tätä osiota ei näy.</p>
<p>Komentoriviltä <code>blunderdb info --db tiedosto.db</code> näyttää alkuperän ja allekirjoituksen tilan <strong>kirjoittamatta koskaan tiedostoon</strong>. Komento toimii myös suojatulle tiedostolle ilman salasanaa. Ks. <code>CLI_USAGE.md</code> komennon <code>export</code> valitsimista <code>--watermark</code> ja <code>--password</code> sekä komennoista <code>identity</code> ja <code>open</code>.</p>
<h4>Tietokannan julkaiseminen muille</h4>
<p>Merkitty tietokanta jaetaan kuin mikä tahansa tiedosto — sähköpostilla, omalla sivustolla, USB-tikulla. blunderDB <strong>ei tarjoa mitään palvelua</strong>: ei varastoa, ei isännöityä luetteloa, ei tiliä. Se seuraa suoraan sen rakenteesta: tiedoston vastaanottajan puolella ei koskaan kirjata mitään, joten palvelulle ei olisi mitään ilmoitettavaa, vaikka sellainen olisi.</p>
<p>Se, mikä tekee julkaistusta tietokannasta toisen käytettävän, tiivistyy neljään kenttään, jotka ovat kaikki jo olemassa:</p>
<ul>
<li><strong>Käyttäjä</strong> — kuka sen kokosi, sillä nimellä, jonka haluat mainittavan.</li>
<li><strong>Kuvaus</strong> — mitä tietokanta sisältää, yhdessä lauseessa, joka mahtuu luetteloon: «240 tuplauspäätöstä tilanteessa, kommentoituna, keskitaso».</li>
<li><strong>Alkuperä</strong> (vesileimasta) — mikä tämä tiedosto on ja kenelle se tuotettiin. Se on ensimmäinen asia, jonka vastaanottaja lukee <em>Metatiedot</em>-paneelista.</li>
<li><strong>Myöntäjän sormenjälki</strong> — julkaise se tiedoston vierellä, ei sen sisällä: sitä vertaamalla vastaanottaja varmistaa, että tiedosto tulee sinulta eikä joltakulta, joka on ottanut nimesi.</li>
</ul>
<p>Ilman vesileimaa julkaistu tietokanta on täysin käyttökelpoinen; se on vain nimetön, eikä <em>Metatiedot</em>-paneeli näytä silloin <em>Alkuperä</em>-osiota.</p>
<p>Tietokannan tunnetuksi tekemiseen <code>varaston keskustelujen &lt;https://github.com/kevung/blunderDB/discussions&gt;</code>_ <em>Show and tell</em> -kategoria toimii hakemistona: se on julkaisijoiden ylläpitämä lista, ei blunderDB:n tarjoama palvelu. Sinne ilmoittaminen vaatii linkin, yllä olevat neljä kenttää ja sormenjäljen.</p>
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
<td>log</td>
<td>Avaa toimintalokin: lokitiedoston kaksisataa viimeistä riviä, sekä keinot kopioida ne raporttiin tai avata ne sisältävä kansio.</td>
</tr>
<tr>
<td>ask</td>
<td>Kääntää sanallisen lauseen — ranskaksi tai englanniksi — hakutunnuksiksi: <code>ask my cube blunders at a score</code>. Tunnukset kirjoitetaan komentoriville, niitä ei suoriteta: lue ne ja paina sitten Enter. Se mitä ei ymmärretty, sanotaan, ei koskaan arvata.</td>
</tr>
<tr>
<td>like</td>
<td>Korvaa selatun listan asemilla, jotka ovat lähimpänä nykyistä — tai sitä, jonka indeksi annetaan (<code>like 42</code>). Läheisyys on kuljetusetäisyys nappulapipeinä: se ei ole suodatin, se järjestää koko tietokannan sen sijaan että rajaisi sitä, eikä siksi yhdisty hakutunnuksiin.</td>
</tr>
<tr>
<td>train</td>
<td>Aloittaa mikroharjoitusistunnon. Ottaa argumentin: <code>train pips</code> (pip-laskenta), <code>train epc</code>, <code>train tp</code> (hyväksymispiste ottelutilanteessa), <code>train quiz</code> (siirto tai kuutiopäätös, arvosteltuna tallennettua analyysiä vasten). Viisi kysymystä, ajastettuna, heti korjattuna.</td>
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
<td>Tuo yhden tai useamman aseman/ottelun tiedostosta (xg, xgp, sgf, mat, txt, bgf). Argumentin kanssa — <code>import XGID=…</code> tai <code>import OGID=…</code> — lukee tunnisteen tiedostovalitsimen avaamisen sijaan, kun se tulee viestistä, foorumilta tai skriptistä.</td>
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
<td>gt:holding</td>
<td>Asema kuuluu tiettyyn pelisuunnitelmaan, vuorossa olevan pelaajan näkökulmasta: <code>race</code>, <code>bearin</code> (kotiutus kontaktissa), <code>crunch</code>, <code>backgame</code>, <code>acepoint</code>, <code>blitz</code>, <code>primevprime</code>, <code>mutualholding</code>, <code>holding</code>, <code>contact</code>. Toistettavissa (<code>gt:holding gt:mutualholding</code>). Johdettu merkintä kuten vaihe: laskettu laudasta, ei koskaan muokattavissa, <code>blunderdb repair</code> laskee sen uudelleen.</td>
<td><code>--game-type</code></td>
</tr>
<tr>
<td>#prime</td>
<td>Asema kantaa tätä <strong>tunnistetta</strong> jossakin kommentissaan. Tunniste on proosaan kirjoitettu <code>#sana</code>; mikään ei ilmoita sitä. Vertailu on rajattu, joten <code>#prime</code> ei löydä sanaa <code>#priming</code> — juuri siinä on ero tekstisuodattimeen, joka etsii osamerkkijonoa. Toistettavissa, ja tunnisteet <strong>kasautuvat</strong> (<code>#prime #backgame</code> pyytää molempia): asema kantaa useita tunnisteita, joten kahden nimeäminen tarkoittaa ”molempia”.</td>
<td>—</td>
</tr>
<tr>
<td>n&gt;x</td>
<td>Asema kohdattiin tietokannassa yli x kertaa — niiden siirtojen määrä, jotka johtavat siihen, kaikissa otteluissa. Muodot <code>n&gt;3</code>, <code>n&lt;2</code>, <code>n3,10</code> ja <code>n4</code> (täsmälleen neljä).</td>
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
