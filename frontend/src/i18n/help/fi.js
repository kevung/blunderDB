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
<p>Asemat tallennetaan tietokantaan, jota edustaa <em>.db</em>-tiedosto. Työpöytäsovellus avaa tämän tiedoston suoraan, ei koskaan verkko-osoitetta: palvelintila on saman binäärin toinen tila, ja toisesta toiseen siirrytään viemällä tai migroimalla tietokanta, ei osoittamalla sovellusta URL-osoitteeseen.</p>
<h3>Päätoiminnot</h3>
<p>blunderDB:n keskeiset mahdolliset toiminnot ovat:</p>
<ul>
<li>lisätä uusi asema,</li>
<li>muokata olemassa olevaa asemaa,</li>
<li>kopioida laudan kuva leikepöydälle (PNG) näppäimillä <strong>CTRL-X</strong>, tai täydellisen analyysin kanssa näppäimillä <strong>CTRL-X CTRL-X</strong>,</li>
<li>poistaa olemassa oleva asema,</li>
<li>hakea yksi tai useampi asema,</li>
<li>tuoda otteluita eri lähteistä (XG, GNUbg, BGBlitz, Jellyfish, HedgeHog), mukaan lukien kommentit XG-tiedostoista,</li>
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
<li>harjoitella laskutehtävillä (Harjoittelu-paneeli),</li>
<li>opiskella asemia välitoistolla (Anki-paneeli),</li>
<li>litteroida ottelu käsin (Litterointi-paneeli),</li>
<li>näyttävät tietokannan metatiedot (Metatiedot-paneeli).</li>
</ul>
<p>Paneelin korkeutta säädetään vetämällä sen kahvaa; jokainen välilehti muistaa omansa.</p>
<p>Modaali-ikkunoita voidaan näyttää seuraaviin tarkoituksiin:</p>
<ul>
<li>näyttää blunderDB:n ohje, jonka hakukenttä (<em>/</em> siirtää kohdistimen sinne) valitsee kirjoitetun tekstin esiintymät kirjainkoosta ja aksenteista riippumatta (<em>ENTER</em> siirtyy seuraavaan, <em>MAJ-ENTER</em> edelliseen),</li>
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
<p>Hiiren oikea napsautus laudalla avaa valikon, joka tarjoaa: näytetyn aseman arvioinnin Eval-paneelissa, sen peilikuvan arvioinnin, laudan kuvan ja sen analyysin kopioinnin leikepöydälle (<em>CTRL-X CTRL-X</em>:n vastine, vaikeampi löytää), <strong>kuvan tallennuksen tiedostoon</strong> SVG- tai PNG-muodossa, uuden näkymän avaamisen tähän asemaan, ja — jos asema tulee jo tietokannasta — sen lisäämisen Anki-pakkaan (välein toistaminen) tai sen <strong>naapuriasemien</strong> järjestämisen (ks. Hakupaneeli).</p>
<p>Leikepöytä on arkinen ele; tallentaminen on se toinen tarve — kuvitus artikkeliin, foorumiviestiin, oppituntiin. <strong>SVG</strong> tarjotaan siksi, että lauta on sellainen: se on muoto joka kestää suurentamisen, se jonka voi panna asiakirjaan sumentumatta. PNG johdetaan siitä, kuten leikepöydälle kopiointikin: yksi renderöinti, kolme määränpäätä, joten yksikään ei voi ajautua muista erilleen. Tämä valikko ei ilmesty Eval-paneelissa eikä Haku-paneelissa, joissa oikea painike jo asettaa toisen värin nappuloita. Katso Aseman tuominen Eval-paneeliin aseman tuomisesta Eval-paneeliin.</p>
<p>Tilarivi on jäsennelty vasemmalta oikealle seuraavin tiedoin:</p>
<ul>
<li>komentorivi, joka avataan painamalla <em>VÄLILYÖNTI</em>-näppäintä,</li>
<li>käyttäjän suorittamaan toimintoon liittyvä tiedotusviesti,</li>
<li>nykyisen aseman järjestysnumeron, jota seuraa asemien määrä nykyisessä kirjastossa (tai siirto-/pelitiedot ottelua selattaessa),</li>
<li><strong>kirjastolaskurin</strong> — ”412 asemaa · 38 blunderia · 5 ottelua” — jossa jokainen luku <strong>avaa sen, mitä se laskee</strong>: asemat, komentoriville kirjaston kynnyksellä valmistellun <code>E&gt;</code>-haun tai otteluluettelon. Luku, jota ei voi seurata, on koriste. Blunderin kynnys on kirjaston oma, säädetty asetusten <em>Kirjasto</em>-välilehdellä ja jaettu tilastojen kanssa: kaksi kynnystä saisi saman sanan tarkoittamaan kahta asiaa. Laskuri lupaa täsmälleen sen, minkä linkki avaa, myös sellaisen aseman osalta, joka on pelattu useilla eri tavoilla ja joka on suurimman kustannuksensa arvoinen. Hyvin suuressa kirjastossa (yli 200 000 riviä) laskuri ei käy taulukoita läpi: ”≈”-merkillä alkava luku on arvio (yläraja, koska poistetut rivit jättävät aukkoja). Jos positiot ovat arvioita, blunderien määrä, jolle ei ole rehellistä arviota, näkyy muodossa ”?” — linkki käynnistää haun, joka antaa tarkan määrän.</li>
</ul>
<div class="admonition note">
<p>Käyttäjän haun tuloksena saaduissa asemissa tilarivillä näkyvä asemien määrä vastaa suodatettujen asemien määrää.</p>
</div>
<p><strong>Anki</strong>-välilehti kantaa <strong>merkkiä</strong>, kun kortteja on kerrattavana, kaikki pakat mukaan lukien. Tuo luku on syy avata välilehti; sillä ei ole asiaa sen taakse. Nolla ei näytä mitään: ”0”:aa näyttävä merkki on kohinaa.</p>
<p>Komento <code>log</code> avaa <strong>toimintalokin</strong>: lokitiedoston kaksisataa viimeistä riviä, painikkeen niiden kopioimiseen — juuri sen mitä raportin liittäminen ilmoitukseen vaatii — ja toisen kansion avaamiseen. Lokia ei suodateta eikä muotoilla uudelleen: siistitty loki ei enää kelpaa lainaukseksi.</p>
<p>Komento <code>grid</code> avaa <strong>vedosarkin</strong>: selattavan luettelon — haun tulokset, kirjasto, kokoelma — pienoislautojen ruudukkona, piirrettyinä kuten lauta, kahdenkymmenenneljän sivuina. Se avautuu nykyisen aseman sivulle, ja aseman pienoiskuva on kehystetty; napsautus tai <em>ENTER</em> pienoiskuvan kohdalla avaa sen aseman laudalle ja sulkee sarkin, ja sitä voi selata kokonaan näppäimistöllä (katso Vedosarkki). Se ei avaudu muokkaustilassa eikä ottelussa, jota selataan sen siirtojen mukaan.</p>
<p>Hakupaneelin <strong>hakuhistoriassa</strong> tallennetun komennon jokainen tunnus näkyy nimettynä merkkinä — <em>Ei kontaktia</em>, <em>Siirtovirhe</em> — paljaan tunnuksen sijaan. Tarkka komento jää työkaluvihjeeseen, sillä juuri se ajetaan uudelleen; ja tunnus jota blunderDB ei tunnista näkyy <strong>sellaisenaan</strong> eikä lähimmäksi käännettynä.</p>
<h3>Näkymävälilehdet</h3>
<p>Työkalupalkin alla oleva välilehtipalkki mahdollistaa työskentelyn useiden <strong>näkymien</strong> kanssa rinnakkain. Jokainen näkymä on itsenäinen työtila, joka säilyttää oman asemaluettelonsa, nykyisen aseman indeksin, näytetyn aseman, analyysin ja valitun siirron, aktiivisen paneelin, käynnissä olevan kommentin sekä ottelun navigointikontekstin. Näin voi esimerkiksi pitää haun auki yhdessä näkymässä ja selata samalla ottelua toisessa.</p>
<ul>
<li><strong>Näkymän luominen</strong>: napsauta välilehtipalkin <em>+</em>-painiketta (nimeltään <em>+ Uusi näkymä</em>, kunhan näkymiä on vain yksi) tai paina <em>CTRL-T</em>. Uusi näkymä käynnistyy nykyisen näkymän kopiona.</li>
<li><strong>Näkymän sulkeminen</strong>: napsauta välilehden ruksia tai paina <em>CTRL-W</em>. Viimeistä näkymää ei voi sulkea.</li>
<li><strong>Näkymän vaihtaminen</strong>: napsauta välilehteä, paina <em>CTRL-PageUp</em> / <em>CTRL-PageDown</em> (tai <em>SHIFT-J</em> / <em>SHIFT-K</em>) siirtyäksesi edelliseen / seuraavaan näkymään, tai <em>CTRL-1</em> – <em>CTRL-9</em> siirtyäksesi suoraan n:nteen näkymään.</li>
<li><strong>Näkymän uudelleennimeäminen</strong>: kaksoisnapsauta välilehteä, kirjoita uusi nimi ja vahvista painamalla <em>ENTER</em>.</li>
</ul>
<p>Näkymät tallennetaan tietokannan istuntotilan mukana ja palautetaan sen uudelleenavauksen yhteydessä.</p>
<h3>Asetukset</h3>
<p>Työkalurivin asetuspainike (rataskuvake), ohjepainikkeen vasemmalla puolella, avaa blunderDB:n asetusikkunan. Se on jaettu yhdeksään välilehteen:</p>
<ul>
<li><strong>Käyttöliittymä</strong> — teema, kieli, näytön skaalaus, paneelin sijainti, PageUp / PageDown -askel (10, 50, 100, 500 tai 1 000 asemaa tai 10 % luettelosta), lokit, päivitysten tarkistus, oma nimesi ja naapuriasemat;</li>
<li><strong>Laudan värit</strong> — laudan värit;</li>
<li><strong>Kirjasto</strong> — se, mikä kuuluu avoinna olevaan tietokantaan: virheen ja blunderin kynnykset sekä tiivistys ja korjaus, jotka kuvataan alla;</li>
<li><strong>Korpus</strong> — kaksoiskappaleet tuonnissa, pelaajien ja tapahtumien aliakset sekä todennäköisten kaksoiskappaleiden haku (katso otteluiden tuonti);</li>
<li><strong>Bearoff</strong> — Eval-paneelin käyttämät ulosmenotaulukot;</li>
<li><strong>gammonNet</strong> — sisäänrakennetun evaluaattorin asetukset, kuvattu alla;</li>
<li><strong>Valvottu kansio</strong> — kansioon saapuvien otteluiden automaattinen tuonti, kuvattu alla;</li>
<li><strong>Avustaja ja MCP</strong> — paikallinen MCP-palvelin ja sisäinen avustaja, kuvattu alla;</li>
<li><strong>Merkitsijän identiteetti</strong> — avain, jolla alkuperämerkintäsi allekirjoitetaan; kuvattu luvussa Tietokannan jakaminen: alkuperä ja salasana.</li>
</ul>
<p><em>Käyttöliittymä</em>-välilehti alkaa <strong>teemalla</strong>: <em>seuraa järjestelmää</em>, <em>vaalea</em>, <em>tumma</em>, <em>suuri kontrasti</em> tai <em>tulostettava</em>. Teema asettaa käyttöliittymän värit ja <strong>ehdottaa lautapalettia</strong> — tumma käyttöliittymä vaalean laudan ympärillä ei ole tumma teema vaan puolikas, sillä lauta täyttää suurimman osan ikkunasta.</p>
<p>Sinulla on viimeinen sana, ja mekanismi takaa sen sen sijaan että lupaisi: <em>Värit</em>-välilehti säätää edelleen lautaa suoraan, ja teeman jälkeen valittu väri on sinun. Käynnistyksessä sovelletaan vain käyttöliittymän symboleja, ei koskaan lautapalettia — asettamasi on jo ladattu, ja sen ylikirjoittaminen joka käynnistyksessä pyyhkisi työsi istunto kerrallaan. Katso <code>ADR-0038 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0038-a-named-theme-carries-the-board-palette-and-the-user-still-has-the-last-word.md&gt;</code>__.</p>
<p><em>Seuraa järjestelmää</em> on oletus: se noudattaa työpöydän vaalea/tumma-asetusta, myös kun se muuttuu kesken istunnon. Työkalu ei tyrkytä vaaleaansa tai tummaansa työpöydälle joka on jo päättänyt.</p>
<p><em>Käyttöliittymä</em>-välilehdellä voi myös valita kielen: englanti, ranska, saksa, italia, espanja, suomi, japani, kreikka tai venäjä. Koko käyttöliittymä (työkalurivi, paneelit, viestit, ohje) käännetään valitulle kielelle. Kielivalinta tallennetaan ja säilyy istunnosta toiseen.</p>
<p><em>Kirjasto</em>-välilehti kokoaa sen, mikä kuuluu avoinna olevaan tiedostoon eikä koneeseen. Se on tyhjä niin kauan kuin yhtään tietokantaa ei ole avattu, ja se sanoo sen.</p>
<p>Se kantaa ensiksi kahta <strong>kynnystä</strong>, jotka ratkaisevat koko sovelluksen sanaston: päätös on <strong>virhe</strong> heti kun sen kustannus yltää virhekynnykseen, ja tuo virhe on <strong>blunder</strong> heti kun se yltää blunderin kynnykseen. Jokainen blunder on virhe, joten ensimmäinen kynnys ei voi ylittää toista, ja blunderDB kieltäytyy käänteisestä parista. Arvot syötetään equitynä — ”0,080” — kaikkien taulukoiden yksikkönä; komentorivi puolestaan puhuu millipisteinä, joten kynnys 0,080 kirjoitetaan haussa <code>E&gt;80</code>.</p>
<p>Nämä kynnykset seuraavat tiedostoa, eivät tietokonetta: sama tietokanta laskee samat blunderit kaikkialla, missä se avataan, <code>blunderdb info</code> näyttää ne ja <code>blunderdb edit --error-threshold</code> / <code>--blunder-threshold</code> säätää niitä. Ne eivät kulje mukana viennissä: kynnys on lukutapa, ei asemien ominaisuus.</p>
<p>Kolme <strong>esiasetusta</strong> on tarjolla yhdellä napsautuksella, kukin sen ohjelman nimellä, joka on vetänyt tuon rajan: blunderDB (0,050 / 0,100), XG (0,020 / 0,080) ja gnubg (0,040 / 0,080). Oletuksena kirjasto lukee 0,050 ja 0,100.</p>
<p>Mitä ne muuttavat näytöllä: tilarivin laskurin blunderien määrän ja haun, jonka sen linkki valmistelee, tilastojen ja pelaajataulukon sarakkeet ”Virheet” ja ”Blunderit”, siirtojen merkinnät ottelun tietonäkymässä (Ottelupaneeli) sekä luettelon asemista, joita blunderDB tarjoaa katsottavaksi tuonnin jälkeen.</p>
<p>Samalta välilehdeltä löytyy myös painike <strong>Tiivistä tietokanta</strong>, joka ottaa takaisin poistojen (ottelut, turnaukset, siivoukset) jättämän levytilan: tietokanta ei koskaan pienene itsestään dataa poistettaessa, tiivistys on pyydettävä nimenomaisesti. Toiminto voi kestää suuressa tietokannassa. Tiivistetty kopio kirjoitetaan tiedoston viereen ja korvaa sen sitten: tilapäisesti tarvitaan noin tietokannan koon verran vapaata levytilaa sen levyosiolla, ei muistissa. Jos jokin toinen ohjelma pitää tietokannan auki (esimerkiksi toinen, vain luku -tilassa oleva instanssi), tiedostoa ei korvata: tietokanta tiivistetään paikallaan, mikä vaatii noin kaksinkertaisen koon verran vapaata tilaa, ja väliaikainen tiedosto menee järjestelmän väliaikaiskansioon tai ympäristömuuttujan <code>SQLITE_TMPDIR</code> nimeämään kansioon: jos <code>/tmp</code> on pieni, osoita muuttuja tietokannan levyosioon. blunderDB kieltäytyy käynnistymästä ja kertoo viestissä, mitä puuttuu (levytilaa tai alle 512 Mt käytettävissä olevaa muistia), sen sijaan että riskeeraisi keskeytyneen tiivistyksen. Ennen käynnistystä pyydetään vahvistus. Tulos — säästynyt tila megatavuina — näkyy sen jälkeen tilarivillä. Sama toiminto on käytettävissä komentoriviltä komennolla <code>blunderdb vacuum</code> (katso Komentoriviliittymä (CLI)).</p>
<p><em>Käyttöliittymä</em>-välilehden <strong>Avaa lokikansio</strong> -painike avaa kansion, jossa sovelluksen loki sijaitsee — kätevää, kun vikailmoitukseen halutaan liittää yksityiskohtia, erityisesti kun blunderDB on käynnistetty pikakuvakkeesta tai kaksoisnapsautuksella ilman päätettä, joka näyttäisi mitään.</p>
<p>Samaisen välilehden <strong>Tarkista päivitykset käynnistyksessä</strong> -valintaruutu, oletuksena pois päältä, kysyy kerran käynnistystä kohden GitHub-arkiston julkaisusivulta ja näyttää tilarivillä viestin, jos uudempi versio on saatavilla — ei koskaan ikkunaa, joka estäisi työskentelyn. Tarkistus pysyy automaattisesti pois päältä asennuksessa, joka on tehty paketinhallinnan kautta (Flatpak, Homebrew, jakelun paketti…): silloin päivityksistä huolehtii se kanava eikä blunderDB itse.</p>
<p>Kaksi <em>Käyttöliittymä</em>-välilehden asetusta ohjaa <strong>naapuriasemien</strong> järjestämistä (tunnus <code>like</code>, ks. Hakupaneeli): palautettavien naapureiden määrä ja enimmäisetäisyys, jonka takana asema lakkaa olemasta naapuri. Tämä etäisyys on oletuksena nolla, siis ilman kattoa: mittakaava riippuu pelin vaiheesta, ja tässä valittu arvo luettaisiin mittaukseksi. Tunnus <code>like&lt;12</code> asettaa omansa yhtä hakua varten, asetukseen koskematta.</p>
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
<p><em>Kirjasto</em>-välilehti kantaa lopuksi <strong>Korjaa analyysit</strong>: analyysisarakkeet, joita haku ja tilastot kysyvät, ovat projektio tallennetuista analyyseista, jotka pysyvät koskemattomina. Projektion vika on siis korjattavissa ilman uudelleentuontia. Se on nimenomaista eikä koskaan automaattista — jonkun analyysisarakkeiden uudelleenkirjoittaminen pelkästään siksi, että hän avaa tietokantansa, ei ole asia, jonka työkalun tulisi tehdä hänen selkänsä takana. Sama <code>blunderdb repair</code> on käytettävissä komentoriviltä.</p>
<p><strong>gammonNet</strong>-välilehti säätää sisäänrakennettua evaluaattoria (katso <code>ADR-0011 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0011-gammonnet-is-ported-to-go-and-the-representation-boundary-sits-at-the-evaluator-s-edge.md&gt;</code>__). Siinä on kaksi säädettävää hakusyvyyttä, jotka on nimetty ja tallennetaan erikseen — toisen alentaminen ei koskaan muuta toista:</p>
<ul>
<li><strong>Näyttösyvyys</strong> — interaktiivinen mukavuus lautaa muokattaessa; ei koskaan kirjoiteta tietokantaan.</li>
<li><strong>Analyysisyvyys</strong> — se, mitä tuonnin jälkeinen analyysierä kirjoittaa aseman Analyysiin.</li>
</ul>
<p>Molempien oletusarvo on <strong>2-ply</strong>, kanoninen asetus. Välilehdellä voi säätää myös <strong>karsinnan</strong> (oletus <code>k=12</code>) ja <strong>näytettävien ehdokassiirtojen määrän</strong> (oletus 10) sekä valintaruudun <strong>analysoi automaattisesti tuonnin jälkeen</strong>, joka käyttöön otettuna tarkistaa jokaisen tuonnin jälkeen, onko jäljellä asemia <strong>ilman yhtään analyysiä</strong> (ei gammonNet-, XG-, GNUbg- eikä BGBlitz-analyysiä — sääntö on « evaluointi täyttää vain aukon », ei koskaan korvaa), ja tarvittaessa käynnistää taustalla gammonNet-analyysin määritetyllä analyysisyvyydellä. Painike <strong>Analysoi nyt</strong> käynnistää saman täydennysanalyysin uudelleen manuaalisesti — hyödyllinen ennen tätä ominaisuutta luodun kirjaston saattamiseksi ajan tasalle.</p>
<p>Toinen painike, <strong>Analysoi vanhentuneet positiot uudelleen</strong>, kattaa päinvastaisen tapauksen: jo gammonNetin analysoima positio, jonka tallennettu analyysi on kuitenkin kirjoitettu nyt käynnissä olevaa vanhemmalla moottoriversiolla tai eri syvyydellä kuin yllä määritetty analyysisyvyys, merkitään siellä vanhentuneeksi ja arvioidaan uudelleen. Positiota, jolla on lisäksi XG-, GNUbg- tai BGBlitz-analyysi, tämä painike ei koskaan koske, riippumatta sen gammonNet-sisällöstä — <code>ADR-0013 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0013-evaluations-fill-gaps-an-imported-analysis-is-never-overwritten.md&gt;</code>__:n suoja pysyy ehdottomana. Kunkin painikkeen vieressä näkyvä luku (asemat ilman analyysiä, vanhentuneet asemat) on puhtaasti informatiivinen; erä laskee oman listansa uudelleen käynnistyessään.</p>
<p>Molemmat erät ovat <strong>rajattuja, näkyviä ja peruutettavissa, eivät koskaan hiljainen daemon</strong>: niiden edistyminen (<code>analysoidut asemat / yhteensä</code>) ja peruutuspainike näkyvät tilarivillä koko niiden keston ajan ja katoavat, kun ne on valmis, ja niiden tilalle tulee tuloksen yhteenveto — kuinka monta positiota <strong>analysoitiin</strong>, kuinka monta <strong>hylättiin</strong> (positio, jota gammonNet kieltäytyy arvioimasta, kuten sen taulukon kattavuuden ulkopuolella oleva ottelutulos, mikä ei koskaan ole virhe) ja kuinka monta <strong>epäonnistui</strong> (yritetään uudelleen muuttumattomana seuraavalla ajolla). Sovelluksen sulkeminen kummankaan aikana ei hukkaa mitään: jokainen analysoitu positio kirjoitetaan sitä mukaa, ja seuraava ajo jatkaa täsmälleen siitä, mihin analyysi jäi, ilman minkäänlaista lokia.</p>
<p><strong>Tietokannan ottelun pääomataulukko.</strong> Välilehden alareunassa oleva luettelo <strong>Tietokannan ottelun pääomataulukko (MET)</strong> valitsee taulukon, jolla gammonNet arvottaa ottelutilanteet: Kazaross-XG2, sisäänrakennettu ja oletus, tai GNUbg:n <code>.xml</code>-tiedostosta tuotu taulukko painikkeella <strong>Tuo .xml-MET…</strong>. Valinta kuuluu tietokannalle, ei sovellukselle: toisen tietokannan avaaminen palauttaa sen taulukon. Vain eksplisiittiset taulukot luetaan (ei parametrisia <code>zadeh</code>- tai <code>mec</code>-taulukoita); tuodun taulukon pituuden jälkeen sisäänrakennettu taulukko ottaa vastuun. Taulukko tunnistetaan arvoistaan, ei nimestään: GNUbg:n <code>Kazaross-XG2.xml</code>-tiedoston tuonti ei lisää mitään, se on sisäänrakennettu taulukko. Jokainen gammonNetin laskema analyysi tallentaa taulukon, jolla se laskettiin; tuotujen analyysien (XG, GNUbg, BGBlitz) oletetaan laskettu Kazaross-XG2:lla. Ottelutilanteen analyysi, joka on laskettu muulla kuin nykyisellä taulukolla, merkitään analyysipaneelissa merkinnällä <strong>Eri MET</strong> ja jätetään pois tilastoista (virheiden keskiarvot, PR, rankingit, kaksinpelit); rahapelianalyysia ei koskaan. Taulukon vaihtaminen ei kirjoita mitään analyysia uudelleen: vain se muuttuu, mitä vertailut ottavat huomioon.</p>
<p>Paneelin suora arviointi ja tuplausmatriisi arvotetaan tietokannan taulukolla, kuten sen tallentamat analyysit. Taulukko kulkee niiden analyysien mukana, jotka viittaavat siihen: tietokannan vienti ottaa sen mukaan, ja tietokannan tuonti lisää sen vastaanottavaan tietokantaan ilman kaksoiskappaletta (jo olemassa oleva taulukko tunnistetaan sen arvoista) ja tekemättä siitä nykyistä. gammonNet-analyysi, joka korvataan toisen moottorin (XG, GNUbg) analyysilla, katsotaan jälleen lasketuksi Kazaross-XG2:lla.</p>
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
<li><strong>Tuonti on hiljainen.</strong> Tutkit asemaa, kun ottelusi saapuivat: näytön ottaminen sinulta olisi pahin mahdollinen hetki. Tila, aktiivinen haku, välilehti ja näytettävä asema eivät muutu; asemaluetteloa ei ladata uudelleen, ja se näyttää uudet ottelut seuraavan uudelleenlatauksen yhteydessä. Tuonti tehdään ilman ikkunaa, ja tilapalkki näyttää bannerin, jossa on tuotujen, ohitettujen (kaksoiskappaleet) ja epäonnistuneiden otteluiden määrä, sekä painikkeen, joka avaa halutessasi täydellisen raportin. Kaikki muu on samaa kuin käsin tehdyssä tuonnissa: samat kaksoiskappaleet tunnistetaan, sama tuontierä, sama automaattinen analyysi, jos se on käytössä.</li>
</ul>
<p>Oletusväli on kymmenen sekuntia; alaraja kaksi. Kansiota ei käydä läpi rekursiivisesti: valvottu kansio on paikka johon työkalu pudottaa ottelunsa, ei tutkittava puu. Irrotettu verkkojako ei pysäytä valvontaa eikä saa sisältöään käymään uudesta palatessaan.</p>
<p>Sama valvonta on olemassa komentorivillä komennolla <code>blunderdb import --type batch --dir &lt;kansio&gt; --watch</code> (katso Komentoriviliittymä (CLI)): se on muoto, jota palvelin, ajastettu tehtävä tai skripti voi käyttää.</p>
<h4>Avustaja ja MCP</h4>
<p>Välilehti <strong>Avustaja ja MCP</strong> säätää kahta asiaa, jotka kumpikin ovat oletuksena pois päältä.</p>
<p><strong>Paikallinen MCP-palvelin</strong> tarjoaa blunderDB:n työkalut (haku, aseman ja sen analyysin luku, pelaajan tilastot, tietovisat…) Model Context Protocolia puhuvalle avustajalle, kuten Claude Desktopille tai Claude Codelle, niin kauan kuin ikkuna on auki. Se kuuntelee vain tällä koneella osoitteessa <code>http://127.0.0.1:&lt;port&gt;/mcp</code> (oletusportti 8765) ja hylkää verkkosivulta tulevan pyynnön. Se lisää kaksi näyttötyökalua: näkymän avaaminen hakuun ja aseman näyttäminen. Työkalut vain lukevat, paitsi jos <strong>Salli kirjoittaminen</strong> on valittuna: silloin ne voivat tallentaa aseman, kommentoida sitä, luoda ja täyttää kokoelman, arvioida Anki-kortin ja tallentaa rolloutin aseman analyysin viereen; mitään ei poisteta. Ilman avointa ikkunaa <code>blunderdb mcp</code> tarjoaa samat työkalut (katso Komentoriviliittymä (CLI)).</p>
<p><strong>Sisäinen avustaja</strong> on näiden samojen työkalujen asiakas. Mallia ei ole mukana: se käyttää OpenAI-yhteensopivaa palveluntarjoajaa — oletuksena Ollamaa tällä koneella, tai Groqia, OpenRouteria, Geminiä, Anthropicia tai jotakin muuta osoitetta. Etänä on jokainen tämän koneen ulkopuolinen osoite, valitusta palveluntarjoajasta riippumatta: lauseesi ja työkalujen tulokset poistuvat silloin koneelta, välilehti kertoo tämän ja odottaa suostumustasi juuri tälle osoitteelle; toinen osoite kysyy uudelleen. API-avain tallennetaan järjestelmän avainnippuun, ei koskaan tietokantaan tai asetustiedostoon. Ollamalle oletuksena ehdotettu malli, <code>qwen2.5:7b</code>, on lähtökohta, ei suositus: mitään mallisuositusta ei anneta ilman lähdekoodin mukana toimitetun mittauskokeen tulosta.</p>
<p>Kun avustaja on otettu käyttöön, se on Haku-paneelin <strong>Avustaja</strong>-alivälilehti. Lause — ”yli 50 millipisteen virheeni kisassa” — avaa sen mukaan nimetyn uuden näkymän, suorittaa haun siinä ja siirtyy siihen; sen tunnisteet jäävät hakuhistoriaan. Työkalujen palauttama tieto tulee tietokannasta; mallin kirjoittama on merkitty tekstillä <strong>Mallin vapaata tekstiä</strong>. Avustaja ehdottaa muutoksen vain, jos valintaruutu <strong>Anna avustajan ehdottaa muutoksia (jokainen vahvistetaan)</strong> on valittuna — asetus on erillinen paikallisen MCP-palvelimen kirjoittamisesta — ja jokainen sen valmistelema muutos näytetään, ja se tehdään vasta painikkeella <strong>Vahvista</strong>.</p>
<p>Asetusikkuna sisältää myös käyttöliittymän näyttöasetukset. <strong>Käyttöliittymän skaalaus</strong> -liukusäätimellä voi suurentaa tai pienentää kaikkia käyttöliittymän elementtejä, mistä on hyötyä korkean tarkkuuden näytöillä tai luettavuuden parantamiseksi. <strong>Paneelien sijainti</strong> -valikko määrittää, missä paneelit (haku, ottelut, analyysi) näkyvät suhteessa lautaan: <em>alhaalla</em>, <em>sivulla</em> tai <em>automaattinen</em> (sivu valitaan tällöin leveillä näytöillä käytettävissä olevan tilan parempaa hyödyntämistä varten). Muiden asetusten tavoin nämä valinnat säilyvät istunnosta toiseen.</p>
<h3>Opastetut kierrokset ja esimerkkitietokanta</h3>
<p>Käytön aloittamisen helpottamiseksi blunderDB tarjoaa käyttöliittymästä <strong>opastettuja kierroksia</strong>. Kierrosten luettelo avataan työkalupalkista tai komennolla <code>tutorial</code> (alias <code>tour</code>). Käytettävissä on seitsemän kierrosta: yleinen käyttöliittymäkierros sekä asemien hakuun, otteluiden tarkasteluun, turnausten tarkasteluun, Eval-paneeliin, Anki-kertaukseen ja tilastoihin keskittyvät kierrokset. Jokainen kierros korostaa käyttöliittymän asianmukaiset elementit vaihe vaiheelta, avaa matkan varrella sen paneelin, josta se puhuu, ja sen voi toistaa milloin tahansa. Ensimmäisellä käynnistyskerralla yleinen kierros tarjotaan automaattisesti.</p>
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
<h3>Komentopaletti</h3>
<p>Komentopaletti (<em>CTRL-SHIFT-P</em>) löytää likimääräisellä nimellä sen, mitä ei enää tiedä mistä etsiä: komentorivin komennon, välilehden, kirjaston suodattimen tai ottelun. Kirjoitettujen kirjainten on esiinnyttävä järjestyksessä, ei välttämättä vierekkäin, isoista kirjaimista ja aksenteista välittämättä: ”kmtrs” löytää kuutiomatriisin, ”lyon” Lyonin turnauksen ottelut.</p>
<p>Nuolinäppäimet valitsevat, <em>ENTER</em> suorittaa, <em>ESC</em> sulkee. Komento suoritetaan kuin se olisi kirjoitettu; <code>s</code> ja <code>ss</code> avaavat komentorivin suodattimien kirjoittamista varten; suodatin suoritetaan kuin kaksoisnapsautuksella kirjastossa; ottelu avautuu kuin kaksoisnapsautuksella Otteluiden paneelissa.</p>
<p>Kun Direction on auki, paletti lisää siihen turnauksen: pelaajat, pöydät, käynnissä olevat ottelut ja kilpailut (ks. pikahaku).</p>
<h3>Analyysipaneeli</h3>
<p><strong>Analyysipaneeli</strong> (<em>CTRL-L</em>) näyttää nykyisen aseman analyysitiedot, jotka on tuotu lähteistä eXtreme Gammon (XG), GNUbg, BGBlitz tai gammonNet. Se esittää parhaat vaihtoehdot (pelinappulasiirrot tai kuutiopäätökset) niiden ekviteettiarvoineen ja vastaavine virheineen. <em>d</em>-näppäin vaihtaa pelinappulasiirtojen analyysin ja kuutioanalyysin välillä. Ottelussa navigoitaessa todella pelattu siirto korostetaan vaihtoehtojen luettelossa. Näytä tai piilota paneeli painamalla <em>CTRL-L</em> tai suorittamalla komento <code>list</code>.</p>
<p>Taulukoiden alla <strong>lause</strong> kertoo joskus, mitä pelattu päätös maksoi ja miksi: ”Menetät 120 mp: pelattu siirto jättää kolme yksinäistä nappulaa, kun 13/7 8/7 jättää vain yhden.” Se syntyy kuudesta mitattavasta säännöstä — alttiudesta, tehdystä tai menetetystä kotipisteestä, luovutetuista gammon-mahdollisuuksista, turvallisuudesta joka maksaa enemmän kuin tuottaa, ja kuutiovirheen kahdesta suunnasta (tuplaus liian myöhään tai liian aikaisin, liian löysä hyväksyntä tai liian tiukka luovutus).</p>
<p>Tärkein sääntö on <strong>vaikeneminen</strong>: lause ilmestyy vain, kun sääntö pätee varmasti, ja virheestä joka ylittää kynnyksen, josta lähtien moottorit ovat yhtä mieltä siitä että kyseessä on virhe. Muulloin lausetta ei ole — ei tyhjää kehystä eikä ”emme tiedä”. Väärä selitys maksaa enemmän kuin ei selitystä: se opettaa jotain epätarkkaa.</p>
<p>Sama lause seuraa virhettä siellä, missä juuri teit sen: <strong>Anki-kortin</strong> takapuolella paljastetun analyysin alla ja Päätös-harjoituksen <strong>tietovisan arviossa</strong> mp-kustannuksen alla. Samat vaikenemisen säännöt pätevät: oikea siirto tai virhe, jota mikään sääntö ei selitä, ei lisää mitään.</p>
<p>Kun asemaa on arvioinut <strong>useampi moottori</strong>, paneelin yläreunan palkki asettaa ne rinnakkain: yksi rivi moottoria kohden, sen syvyys ja sen vastaus — kuutiotuomio tai sen oma paras siirto. Se kertoo ensin ovatko ne samaa mieltä, ja juuri erimielisyys sen oikeuttaa: ”XG sanoo tuplaus, otto; gammonNet sanoo ei tuplausta” luetaan yhdellä silmäyksellä, siinä missä ennen piti verrata kahta taulukkoa vinottain.</p>
<p>Moottorin paras siirto on <strong>sen moottorin</strong> paras: ehdokaslista on lajiteltu ekviteetin mukaan kaikki moottorit sekaisin, joten sen ensimmäinen alkio ei ole kenenkään paras siirto erityisesti.</p>
<p>Palkki ilmestyy vain kun moottoreita todella on useampi, ja se on olemassa vain tässä paneelissa: Eval-paneeli esittää <strong>yhden</strong> päätöksen, sisäänrakennetun moottorin (<code>ADR-0017 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0017-the-panel-shows-position-facts-plus-the-one-decision-the-board-asks.md&gt;</code>__), eikä vertailulla olisi siellä sijaa.</p>
<p>Siirrot kirjoitetaan niin kuin ne luetaan laudalta, täällä kuten Eval-paneelissakin: vähiten edennyt nappula liikkuu ensin, ja <strong>nappula, joka käyttää useamman nopan peräkkäin, kirjoitetaan vain kerran</strong> — samalla nappulalla pelattu 64 luetaan <code>24/14</code>, ja <code>24/14*</code>, jos se lyö perille tullessaan. Ketjun yksityiskohdat näkyvät vain silloin, kun ne kertovat jotakin lisää: <em>matkan varrella</em> tehty lyönti säilyttää välipisteensä, <code>24/18* 18/14</code>, muuten lyönti pisteessä 18 katoaisi merkinnästä.</p>
<p>Tuodun analyysin ekviteetti noudattaa samaa sääntöä kuin Eval-paneeli: sarake ilmoittaa oman viitekehyksensä, ”Equity (money)” tai ”Equity (match)” analysoidun aseman pistetilanteen mukaan, ei koskaan pelkkää ”Equity”-sanaa kertomatta asteikkoa. Money game -asemassa voimassa olevat <strong>Jacoby</strong>- ja <strong>Beaver</strong>-säännöt näytetään myös pieninä merkkeinä kuutiopäätöstaulukon alla.</p>
<h4>Rolloutit</h4>
<p>Analyysin alla <strong>Analyysi</strong>-paneeli tarjoaa aseman <strong>rollaamista</strong>: sadat pelit pelataan jokaisesta ehdokassiirrosta tai tuplauskuution toiminnosta, jotta voidaan erottaa kaksi vaihtoehtoa, joita suora arviointi tuskin erottaa. Kolme asetusta: <strong>Nopea</strong> (216 peliä, katkaistu 7 puolisiirtoon), <strong>Vakio</strong> (1296 peliä, katkaistu 11 puolisiirtoon) ja <strong>Vapaa</strong>, jossa jokainen parametri on muokattavissa — katkaisu, vähimmäis- ja enimmäispelit (36:n monikertoja), JSD-raja, syvyys (ply), ehdokkaiden määrä, siemen ja työprosessien määrä. Painike <strong>Käynnistä rollout</strong>, paneelin näppäin <em>r</em> tai komento <code>rollout</code> (alias <code>ro</code>) käynnistävät sen; edistymispalkki seuraa pelattuja pelejä, ja <strong>Peruuta</strong> (tai <em>r</em> uudelleen) pysäyttää sen kirjoittamatta mitään.</p>
<p>Tulos <strong>tallennetaan analyysin viereen, ei koskaan sen tilalle</strong>: tuotua analyysia ei muuteta. Jokainen rollout muodostaa lohkon, jossa on kullekin ehdokkaalle equity, 95 %:n luottamusväli, <strong>JSD</strong> (ero parhaaseen siirtoon eron keskihajontoina: rajasta alkaen siirto on ratkaistu eikä sitä enää pelata) ja pelien määrä. Rollout päättyy heti, kun siirrot on erotettu. <strong>Konfiguraatio</strong> — moottori ja parametrien täydellinen allekirjoitus — aukeaa taulukon alle: kaksi samalla allekirjoituksella tehtyä rolloutia antaa samat luvut. Rollout pelaa tuplauskuution peleissään: järjestys on luotettava, absoluuttinen equity hieman vähemmän, minkä lohko muistuttaa. Se ei pelaa beaveria: sen rivi <em>tuplaus, otto</em> on tavallisen oton rivi, myös rahapelin asemassa Beaver-säännön alla, jossa suora analyysi ottaa beaverin huomioon. Asema, joka ei ole tietokannassa, voidaan rollata mutta sitä ei tallenneta.</p>
<p>Painike <strong>Näytettyyn luetteloon…</strong> (tai <code>ro search</code>) rollaa peräkkäin näytetyn luettelon — hakutulokset, ottelu tai kokoelma — asemat, joilla ei vielä ole tätä rolloutia; vahvistus kertoo kokonaismäärän ennen aloitusta. Jokainen asema kirjoitetaan heti valmistuttuaan: peruutus säilyttää tehdyn, ja uusi käynnistys jatkaa siitä, mihin jäätiin. Eteneminen säilyy paneelin sulkemisen yli.</p>
<h3>Kommenttipaneeli</h3>
<p><strong>Kommentit</strong>-paneeli (<em>CTRL-P</em>) näyttää, lisää ja muokkaa nykyiseen positioon liittyviä kommentteja. Positiolla voi olla useita eri henkilöiden kirjoittamia kommentteja: kaikki näytetään ketjuna uusimmasta vanhimpaan, kunkin tekijän nimen kanssa, ja omat kommenttisi ovat ensimmäisinä. XG-tiedostoista tuodut kommentit liitetään automaattisesti vastaaviin positioihin. Paina <em>CTRL-P</em> tai suorita komento <code>comment</code> näyttääksesi tai piilottaaksesi paneelin.</p>
<p>Kommenttisi allekirjoittava nimi asetetaan asetuksissa, <em>Käyttöliittymä</em>-välilehden <strong>Nimesi</strong>-kentässä; jos se on tyhjä, kommenttisi eivät ole allekirjoitettuja. XG-tuonnin kommentit allekirjoitetaan nimellä <code>XG</code>, ottelutiedoston kommentit sen kirjaajan nimellä. Kommentin uudelleenkirjoittaminen allekirjoittaa sen sinun nimelläsi. Haku <code>au"Alice"</code> poimii positiot, joita Alice on kommentoinut; käyttöliittymän ulkopuolella <code>blunderdb comment add --author</code> kirjoittaa allekirjoitetun kommentin ja <code>blunderdb comment list</code> lukee ne takaisin (katso Komentoriviliittymä (CLI)).</p>
<p>Jokainen tiedostosta tullut kommentti kantaa <strong>alkuperämerkintää</strong> (<code>XG</code>, <code>GNU BG</code>, <code>BGF</code>, tai <em>tuotu</em>, kun alkuperää ei koskaan tallennettu). Itse kirjoittamasi kommentit eivät kanna sellaista: se on tavallinen tapaus, ja jokaisen rivin merkitseminen olisi vain kohinaa. Tuodun kommentin muokkaaminen tekee siitä sinun: muokkauksen jälkeen lause on sinun.</p>
<p>Ero näkyy muuallakin: ottelun poistaminen ei enää tuhoa asemaa, johon <strong>sinä</strong> olit kirjoittanut. Lähdetiedostosta poimittu muistiinpano sen sijaan katoaa yhä sen ottelun mukana, joka sen toi.</p>
<h4>Tunnisteet</h4>
<p><strong>Tunniste</strong> on kommenttiin kirjoitettu <code>#sana</code>. Mikään ei ilmoita sitä, mikään taulu ei säilytä sitä, ja se on tarkoituksellista: sanasto on sinun omaa proosaasi, ja ilmoituksen vaatiminen ennen tunnisteen käyttöä muuttaisi tavan paperityöksi.</p>
<p>Puuttui toinen puoli: <strong>nähdä</strong> se sanasto, jonka on itselleen rakentanut, ja napsauttaa tunnistetta sen sijaan että muistelisi miten sen kirjoitti. Komento <code>tags</code> tai kirjoituskentän vieressä oleva <code>#</code>-painike avaa sanastoikkunan: tämän tietokannan tunnisteet, kukin sitä kantavien <strong>asemien määrän</strong> kanssa, napsautettavina vastaavan haun käynnistämiseksi. Listan alla ovat suositellut tunnisteet, joita tämä tietokanta ei vielä käytä — backgammon-kirjallisuudesta poimittu sanasto (<code>#blitz</code>, <code>#prime</code>, <code>#holding</code>, <code>#backgame</code>, <code>#containment</code>, <code>#crunch</code>, <code>#ace-point</code>, <code>#timing</code>…), ehdotettu eikä koskaan pakotettu: listalta puuttuva tunniste on täsmälleen yhtä arvokas kuin listalla oleva.</p>
<p>Kirjoittaessa <code>#</code> ehdottaa niitä tunnisteita, joita <strong>tämä tietokanta</strong> jo käyttää, ja sitten suositeltuja. Juuri se estää kirjoittamasta <code>#back-game</code> yhtenä päivänä ja <code>#backgame</code> seuraavana — mitä mikään muu ei huomaisi.</p>
<p>Tunnistehaku kirjoitetaan komentorivillä muodossa <code>#prime</code>. Se on <strong>rajattu</strong>: <code>#prime</code> ei löydä sanaa <code>#priming</code>, kun taas tavallinen tekstihaku, joka etsii osamerkkijonoa, ei osaa erottaa niitä. Useat tunnisteet <strong>kasautuvat</strong> — <code>s #prime #backgame</code> pyytää asemat, jotka kantavat molemmat — koska asema kantaa useita tunnisteita: kahden nimeäminen voi tarkoittaa vain ”molempia”. Tämä on päinvastoin kuin vaihe- tai alkuperäsuodattimessa, jossa asemalla on vain yksi arvo ja kahden arvon nimeäminen voi tarkoittaa vain ”jompaakumpaa”.</p>
<p>Sama lista saadaan käyttöliittymän ulkopuolella komennolla <code>blunderdb list --type tags</code> (katso Komentoriviliittymä (CLI)).</p>
<h3>Roskakori</h3>
<p>Aseman, kokoelman, kommentin tai Anki-kortin poisto kulkee <strong>roskakorin</strong> kautta: poisto todella tapahtuu, mutta kopio katoavasta säilytetään kolmekymmentä päivää. Komento <code>trash</code> avaa ikkunan, joka listaa ne, kussakin <em>Palauta</em> ja <em>Poista</em>, sekä painikkeen <em>Tyhjennä roskakori</em>.</p>
<p>Palautettu asema tulee takaisin <strong>analyyseineen ja kommentteineen</strong> — sen palauttaminen alastomana olisi palautus vain nimeltään. Se ei palaa vanhalla numerollaan: alkuperäistä riviä ei enää ole, ja blunderDB tallentaa sen uudelleen sormenjälkensä perusteella, mikä takaa ettei kaksoiskappaletta synny mutta antaa sille uuden tunnisteen. Kokoelma palaa listoineen; sen sisältämiä asemia ei koskaan poistettu — kokoelma on näkymä niihin.</p>
<p>Yli kolmekymmentä päivää vanhat poistaa komento <code>vacuum</code>, ei koskaan tietokannan avaaminen: <code>vacuum</code>:in tekemättä jättäminen on kaiken säilyttämistä.</p>
<div class="admonition note">
<p>Roskakori ei matkusta. Vienti ei kanna sitä mukanaan, eikä ottelun poisto laita siihen mitään: ottelun poistoa seuraava orpojen asemien siivous on automaattista siivousta eikä käyttäjän teko — katso säilytyssääntö kohdassa Ottelupaneeli.</p>
</div>
<h3>Hakupaneeli</h3>
<p><strong>Hakupaneeli</strong> (<em>CTRL-F</em> tai <em>TAB</em>) suodattaa asemia vapaasti yhdisteltävien kriteerien mukaan: pelinappularakenne, kuutiopäätöksen tyyppi, virheen suuruus, päivämäärät, tunnisteet jne. <em>TAB</em>-näppäin avaa samanaikaisesti hakupaneelin ja asemaeditorin, jolloin haettava pelinappularakenne voidaan määrittää suoraan laudalla.</p>
<p>Suodattimet asetetaan <strong>Ehdot</strong>-alivälilehdellä. Kun haku ei löydä mitään, paneeli kertoo sen (”Mikään asema ei täsmää”) ja tarjoaa <strong>Tyhjennä suodattimet</strong> -painikkeen, eikä vain tilarivi.</p>
<p>Hae näytetyistä asemista komennolla <code>ss</code>, jota seuraavat suodattimet (esim. <code>ss nc</code>, <code>ss E&gt;40</code>). <code>ss</code> hakee näytöllä olevasta luettelosta: edellisen haun tuloksista, avoimesta kokoelmasta tai läpikäytävän ottelun asemista, kirjoitettiinpa komento suoraan tai hakupaneelista (<em>TAB</em>). Paneelin valintaruutu <em>Hae nykyisistä tuloksista</em> noudattaa samaa sääntöä; valintaruutu <em>Avaa uudessa välilehdessä</em> näyttää tulokset uudessa näkymässä (katso Näkymävälilehdet) nykyisen näkymän sijaan. Kokoelmassa ja ottelussa <code>s</code> hylätään: se hakisi koko kirjastosta ja korvaisi näytetyn luettelon.</p>
<p>Kokoelmasta tai ottelusta käynnistetyn <code>ss</code>-haun tuloksista poistutaan <em>Esc</em>-näppäimellä yhdellä painalluksella heti, kun kentällä tai kohdistetulla paneelilla ei ole mitään suljettavaa (esimerkiksi analyysissä valittu siirto): blunderDB palaa koko kokoelmaan tai ottelun tarkasteltuun siirtoon ja jätettyyn asemaan. Tämä paluu seuraa vain <code>ss</code>-komentoa: kokoelman tai ottelun päälle avatusta hakupaneelista käynnistetty <code>s</code> hakee koko kirjastosta, eikä <em>Esc</em> enää palaa jätettyyn luetteloon.</p>
<p>Suuren tietokannan haku näytetään ennen kuin se on laskettu: ensimmäinen tulossivu tulee heti näkyviin, ja tilarivi näyttää « Haetaan… » ja kuluneen ajan, kunnes kokonaismäärä tiedetään; se korvaa silloin luettelon alustavan pituuden. Kerrallaan on käynnissä vain yksi haku: uuden haun aloittaminen hylkää edellisen. <em>Esc</em> keskeyttää käynnissä olevan haun ja pysäyttää sen tietokannan läpikäynnin; jos ensimmäinen sivu oli jo näkyvissä, se jää yksin näkyviin, ja tilarivi kertoo siitä.</p>
<p>Paneeli tarjoaa nimenomaisen hallinnan haettavalle <strong>päätöstyypille</strong>: <em>Indifférent</em> (ei suodatinta), <em>Siirto</em> (siirtopäätökset) tai <em>Tuplaus</em> (kuutiopäätökset). Kun <em>Tuplaus</em> on valittuna, toinen luettelo tarkentaa alityypin: <em>Kaikki</em>, <em>Tuplaus / Ei tuplausta</em> (vuorossa olevan pelaajan on päätettävä, tuplaako) tai <em>Hyväksy / Luovuta</em> (vastaus vastustajan tuplaukseen). Hallinta on synkronoitu laudan kanssa: noppien tai kuution muuttaminen laudalla päivittää päätöstyypin ja päinvastoin. <em>Hyväksy / Luovuta</em> -tilassa kuutio näytetään laudan keskellä tarjotulla arvolla; tämä arvo on edelleen muokattavissa.</p>
<p><strong>Pelin vaihe</strong> — avaus, keskipeli, kilpajuoksu, nappuloiden poisto — on merkintä, jonka blunderDB laskee pelkästä laudasta. Sitä ei voi koskaan muokata, ja se on haettavissa komentorivin <code>ph:</code>-merkinnällä (<code>ph:race</code>, toistettavissa: <code>ph:race ph:bearoff</code>). Kolme sen neljästä rajasta ovat ne, joilla GNU Backgammon ohjaa verkkojaan; neljäs, jossa avaus päättyy, on blunderDB:n käytäntö: asema on yhä avauksessa niin kauan kuin kumpikaan puoli ei ole siirtänyt yli neljää nappulaa lähtöpisteiltään, mitään ei ole poistettu eikä mikään ole palkissa.</p>
<div class="admonition note">
<p>Merkinnän laskee uudelleen komento <code>blunderdb repair</code>. Tietokannassa, joka avataan ensimmäistä kertaa tällä versiolla, laskenta tehdään kerran, avattaessa. Tietokanta, jonka vaiheita ei ole koskaan laskettu, ei palauta mitään <code>ph:</code>-haulle — ei mitään, väärän vastauksen sijaan.</p>
</div>
<p>Tunnus <code>like</code> <strong>järjestää</strong> sen sijaan että rajaisi: sen läsnäolo järjestää tuloksen kasvavan etäisyyden mukaan kohdeasemasta — <code>like</code> nykyisestä, <code>like42</code> siitä, jonka indeksi on 42 — ja muut tunnukset rajaavat näin järjestettyä joukkoa, joten <code>s like42 E&gt;80</code> luetaan ”aseman 42 naapurit, joissa mokasin”. Etäisyys on kuljetusetäisyys nappulapipeinä, se määrä nappuloiden liikettä, joka erottaa kaksi asemaa, vuorossa olevan pelaajan näkökulmasta.</p>
<p>Naapuri on sama <strong>ongelma</strong>, ei sama kuvio: järjestys otetaan kohteen luokan sisältä — sama päätöksen tyyppi, sama pelimuoto (raha tai ottelu) kuutiopäätöksessä, ja eri ottelu kuin sen oma, sillä asemat, jotka ympäröivät sitä sen omassa pelissä, ovat sen lähimmät rakenteet olematta koskaan sen naapureita. Nopat, tulos ja kuutio jäävät luokan ulkopuolelle; tavalliset tunnukset rajaavat niillä silloin kun halutaan. <code>like42*</code> laajentaa luokan kaikkiin päätöstyyppeihin ja molempiin pelimuotoihin, ei koskaan kohteen otteluun; <code>like&lt;12</code> hylkää kaiken yli kahdentoista nappulapipin päässä olevan. Järjestys, joka ei löydä mitään, palauttaa tyhjän listan ja sanoo sen, eikä kymmentä asiaankuulumatonta asemaa.</p>
<p><strong>Muokkaustilassa</strong> <code>s like</code> ottaa kohteekseen <strong>piirretyn</strong> laudan: piirretään suunnilleen se asema, jonka muistaa, käynnistetään, ja kirjasto vastaa — siinä missä rakenteen mukainen haku vaatii tarkan piirroksen. Lauta luetaan silloin asemana eikä kuviona: tyhjäksi jätetty piste lasketaan ulos kannetuiksi nappuloiksi, mikä on oikein todelliselle asemalle ja vääristää laskennan puolittain jätetylle piirrokselle.</p>
<p>Jokainen naapuri kantaa etäisyytensä analyysitaulukoiden alla, yhdessä sen aseman kanssa, jota se on lähellä. Juuri se tekee mahdolliseksi arvioida, katsooko naapuria vai sattumaa, ja siinä on katon tarkoitus. Järjestyksen voi käynnistää myös ilman komentoriviä: <em>CTRL-SHIFT-L</em> tai laudan pikavalikon kohta <strong>Naapuriasemat</strong>.</p>
<p>Merkki <code>n</code> laskee <strong>kohtaamiset</strong>: <code>n&gt;3</code> säilyttää paikat, jotka on kohdattu tietokannassa vähintään kolme kertaa, kaikki ottelut ja kaikki pelaajat yhteensä. Se on eri kysymys kuin ”mitä missasin” — paikka, joka on kohdattu kaksikymmentä kertaa ja pelattu hyvin yhdeksäntoista, on yhä sellainen, joka kannattaa osata ulkoa. Lasketaan päätökset, ei otteluita: sama paikka kahdesti samassa ottelussa lasketaan kahdeksi, koska päätöksiä oli kaksi. Pelaajasuodattimen kanssa lasketaan vain kyseisen pelaajan esiintymät: <code>n&gt;3 pl!"Alice"</code> säilyttää paikat, jotka Alice joutui pelaamaan vähintään kolme kertaa, ja <code>pl"Alice"</code> ne, jotka olivat hänen pelaamissaan otteluissa; <code>op"Bob"</code> rajaa laskennan vastaavasti Bobia vastaan pelattuihin otteluihin.</p>
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
<p>Hakupaneelissa on vasemmassa reunassaan kolme välilehteä: <em>Kriteerit</em> (suodattimet), <em>Historia</em> ja <em>Tallennetut</em> — sekä neljäs, <em>Avustaja</em>, kun sisäinen avustaja on käytössä. <strong>Historia</strong>-välilehti luettelee aiemmat haut päivämäärineen ja komentoineen: napsautus valitsee haun ja näyttää siihen liittyvän aseman laudalla, kaksoisnapsautus suorittaa sen uudelleen. Kunkin merkinnän voi tallentaa suodatinkirjastoon (kirjanmerkkikuvake, antamalla suodattimelle nimi) tai poistaa. <strong>Tallennetut</strong>-välilehti sisältää <strong>suodatinkirjaston</strong>: kaksoisnapsauta tallennettua suodatinta suorittaaksesi vastaavan haun uudelleen (katso Liite: Suodattimien edistynyt käyttö). Komento <code>history</code> (alias <code>hi</code>) avaa hakupaneelin.</p>
<p>Kirjaston suodattimen tähti <strong>kiinnittää</strong> sen. Kiinnitetyt suodattimet näkyvät merkkeinä paneelin yläreunassa, mikä tahansa välilehti on auki, numeroituina kirjaston järjestyksessä: merkin napsautus käynnistää suodattimen, ja <code>ALT-1</code> … <code>ALT-9</code> käynnistävät kyseisen sijan kiinnitetyn suodattimen miltä tahansa näytöltä tilassa NORMAL tai EDIT avaamatta paneelia. Suodatin esittää silloin saman kysymyksen kuin kaksoisnapsautuksella, nappularakenne mukaan lukien. Kiinnitys kuuluu tietokantaan: se seuraa uudelleennimettyä suodatinta, katoaa poistetun suodattimen mukana eikä kulje kirjaston viennin mukana.</p>
<p>Uudelleen ajettu haku säilyttää järjestyksensä: <code>s like42</code> järjestää suhteessa asemaan 42 ja <code>s like</code> suhteessa haun mukana tallennettuun lautaan — siihen, jota selattiin tai joka piirrettiin. Merkintää, johon lautaa ei tallennettu, ei ajeta näytöllä olevaa lautaa vastaan, ja tilarivi kertoo sen.</p>
<div class="admonition tip">
<p>Katso saatavilla olevien suodattimien luettelo kohdasta komentoluettelo.</p>
</div>
<h3>Kokoelmapaneeli</h3>
<p>Kokoelmat-, Turnaukset-, Anki- ja Litterointi-paneeleissa otsikon <strong>+</strong>-painike, jota seuraa luotavan nimi (<strong>+ Uusi kokoelma</strong>, <strong>+ Uusi turnaus</strong>, <strong>+ Uusi pakka</strong>, <strong>+ Uusi litterointi</strong>), on ainoa luontitapa: se avaa syöttökentän, jonka <em>Esc</em> tai <strong>Peruuta</strong> sulkee Kokoelmat- ja Turnaukset-paneeleissa. Otteluluettelossa ⌨-kuvake avaa ottelun litteroinnin ja ✎-kuvake korjaa sen metatiedot.</p>
<p><strong>Kokoelmat</strong>-paneeli (<em>CTRL-B</em>) hallinnoi asemakokoelmia. Kokoelmia voi luoda, nimetä uudelleen ja poistaa. Niihin voi lisätä asemia tai poistaa niitä (<em>Del</em>-näppäin, vahvistus pyydetään). Kaksoisnapsauta kokoelmaa selataksesi sen asemia <em>VASEN</em>- ja <em>OIKEA</em>-näppäimillä. Komento <code>ss</code> hakee avoimen kokoelman asemista; <em>Esc</em> palaa sen jälkeen kokoelmaan (katso Hakupaneeli). Kokoelmien ja kokoelman sisäisten asemien järjestystä voi muuttaa vetämällä ja pudottamalla. Paina <em>CTRL-B</em> tai suorita komento <code>collection</code> näyttääksesi tai piilottaaksesi paneelin.</p>
<p>Kokoelma voi olla <strong>elävä</strong>: sen sisältö ei ole enää käsin tehty lista vaan <strong>haun</strong> tulos, joka lasketaan uudelleen joka avauksella. Kokoelman otsikon ◇-painike tekee siitä elävän viimeisimmällä haulla; ◈ kertoo sen jo olevan, ja sama painike palauttaa listan. Mitään ei tuhota: sen sisältämät asemat ovat yhä tallella, kun palaat.</p>
<p>Elävässä kokoelmassa näkyvä painike ❄ <strong>jäädyttää</strong> sen: haun sillä hetkellä valitsemista asemista tulee tavallisen kokoelman sisältö haun järjestyksessä, ja kysely tyhjennetään. Asemat, jotka kokoelmassa oli ennen sen muuttumista eläväksi, korvataan.</p>
<p>Elävä kokoelma, jonka kysely sisältää tunnuksen jota tämä versio ei enää tunne, <strong>kieltäytyy avautumasta</strong> ja sanoo sen sen sijaan että palauttaisi koko tietokannan. Se on ainoa vika, jota tallennetulla suodattimella ei saa olla: laajeta hiljaisuudessa.</p>
<h4>Oppitunnit</h4>
<p><strong>Oppitunti</strong> on vaiheiden sarja, jonka valmentaja kirjoittaa kerran oppilasta varten ja luovuttaa tälle tietokantatiedostossa (katso komento <code>lesson export</code> sivulla cli). Jokaisella vaiheella on otsikko ja teksti, ja se voi näyttää kokoelman, aseman, molemmat tai ei kumpaakaan. Komento <code>le</code> luettelee tietokannan oppitunnit tilarivillä; <code>le 2</code> avaa oppitunnin 2; <code>le edit</code> avaa oppituntieditorin.</p>
<p>Laudan yläpuolelle ilmestyy silloin <strong>lukupalkki</strong>: oppitunnin nimi, vaiheen numero, otsikko ja sitten teksti. <em>Edellinen</em> ja <em>Seuraava</em> vaihtavat vaihetta; vaihe tuo laudalle kokoelman tai aseman, jonka se näyttää, ja sitä selataan tavallisin elein. <em>Sulje</em> poistuu oppitunnilta. Vaihe, jonka kokoelma tai asema on poistettu, säilyttää tekstinsä.</p>
<p>Palkin <em>Vaihe tehty</em> -valintaruutu merkitsee nykyisen vaiheen tehdyksi; toinen napsautus poistaa merkin, ja palkki laskee tehdyt vaiheet. Se on ainoa ele, joka kirjoittaa jotakin: oppitunnin lukeminen, vaiheen vaihtaminen tai oppitunnin avaaminen ei tallenna mitään, ei saavutettua vaihetta eikä avaamista. Merkki kirjoitetaan avattuun tietokantaan, oppilaan omaan, eikä mikään vienti vie sitä mukanaan. Oppitunnin sisältävän tiedoston tuonti luo oppitunnin; samanniminen jo olemassa oleva oppitunti jää koskematta.</p>
<p><strong>Oppituntieditori</strong> avautuu komennolla <code>le edit</code> (<code>le edit 2</code> oppitunnille 2) tai lukupalkin <em>Muokkaa</em>-painikkeella. Vasemmalla ovat oppituntien luettelo ja kenttä uuden luomiseen; oikealla valitun oppitunnin nimi ja kuvaus, sitten sen vaiheet. Jokaisella vaiheella on otsikko, teksti, luettelosta valittu kokoelma ja asema: <em>Nykyinen asema</em> liittää siihen laudalla näkyvän aseman, <em>Irrota asema</em> poistaa sen. <em>Tallenna vaihe</em> kirjoittaa sen muutokset; nuolet siirtävät sitä yhden askeleen; <em>Lisää vaihe</em> lisää uuden loppuun. <em>Lue</em> sulkee editorin ja avaa oppitunnin sen ensimmäisestä vaiheesta; <em>Poista</em> poistaa oppitunnin ja sen vaiheet koskematta kokoelmiin tai asemiin, joita ne näyttivät. Oppitunteja voi luoda ja muokata myös komentoriviltä tai API:n kautta (Oppitunnit).</p>
<p>Varmuuskopio — koko kirjaston vienti vientiikkunasta tai komentoriviltä — sisältää kaikki oppitunnit. Vientiikkunassa <em>Sisällytä oppitunnit</em> -valintaruutu valitsee ne yksitellen: jokainen valittu oppitunti lähtee mukaan niiden kokoelmien ja asemien kanssa, joita sen vaiheet näyttävät, ja tiedostoon voi lisätä alkuperämerkinnän tai suojata sen salasanalla (<code>.dbx</code>) kuten minkä tahansa viennin. Ilman tätä ruutua osittainen vienti (valinta asemista, kokoelmista tai otteluista) ei sisällä oppitunteja.</p>
<h3>Tuonti: mitä kirjoitetaan ja mitä ei koskaan</h3>
<p>Ottelun, aseman tai toisen tietokannan tuonti lisää sen, mikä puuttuu; se ei korvaa sitä, mikä on jo olemassa.</p>
<ul>
<li><strong>Asemaa ei koskaan kahdenneta.</strong> Sen tunnistaa sen identiteetti — nappulat, kuutio, nopat, pistetilanne — ei koskaan tiedosto, josta se tulee: sama asema kahdessa eri ottelussa pysyy yhtenä ainoana rivinä.</li>
<li><strong>Yksi analyysi moottoria kohti.</strong> eXtreme Gammon, GNUbg, BGBlitz ja sisäänrakennettu evaluaattori elävät rinnakkain samassa asemassa, ja Analyysi-paneeli kertoo kunkin alkuperän. Yhden tuonti ei pyyhi toista pois.</li>
<li><strong>Tuotua analyysiä ei koskaan lasketa uudelleen.</strong> blunderDB tallettaa sen sellaisenaan, tasomerkintöineen (”3-ply”, ”XG Roller++”, ”Book”), ekviteetteineen, virheineen, todennäköisyyksineen ja heiton tuureineen. Sääntö kuuluu: ”arviointi täyttää vain aukon” — tuonnin jälkeinen automaattinen analyysi käy läpi vain ne asemat, joilla ei ole <strong>yhtään</strong> analyysiä, ja <em>Analysoi vanhentuneet asemat uudelleen</em> jättää koskematta jokaiseen asemaan, jolla on tuotu analyysi (katso Asetukset).</li>
<li><strong>Jo tallennetun ottelun tuominen uudelleen tuo vain syvemmän.</strong> Ottelu tunnistetaan pelistään (pelaajat, pituus, nopat, siirrot, kuutio), ei analyysistään: yhtään ottelua, peliä tai siirtoa ei kirjoiteta uudelleen. Lähdeohjelmassa asetetut merkinnät lisätään, ja tallennettua syvempi analyysi korvaa sen asema asemalta — saman ottelun Roller++-versio korvaa 3-ply-version tuontijärjestyksestä riippumatta; samalla tai pienemmällä syvyydellä tallennettu analyysi säilyy. Saman tiedoston tuominen uudelleen ei siis kirjoita mitään uudelleen. Tuontiraportti erottaa kaksoiskappaleet, jotka eivät tuoneet mitään, niistä, jotka syvensivät analyyseja. Komentorivillä <code>--skip-duplicates</code> ohittaa kaksoiskappaleen ottamatta siitä muuta kuin merkinnät. Katkaistu ja myöhemmin täydennetty ottelu (enemmän pelejä) ei ole sama ottelu: se tuodaan toisena otteluna.</li>
<li><strong>Lukukelvoton analyysi ei estä ottelun tuontia.</strong> Päätös, jonka analyysissä on arvo, joka ei ole äärellinen luku (NaN tai ääretön, joita joissakin XG-tiedostoissa on), tuodaan ilman tätä analyysiä; muu ottelu tuodaan tavalliseen tapaan, ja tuontiraportti laskee kyseiset päätökset.</li>
<li><strong>Jo toisilla nimillä olemassa oleva ottelu raportoidaan, mutta sitä ei koskaan yhdistetä.</strong> Kaksi ottelun sormenjälkeä sisältävät pelaajien nimet: "Martin A." ja "Alice Martin" ovat kaksi ottelua. Tuonti vertaa myös noppia (pituus, alkutilanne, kunkin pelin nopat) ja raportoi tiedoston rivin alla ja raportissa tietokannassa jo toisilla nimillä olevan ottelun, jonka nopat sillä on. <code>blunderdb repair --duplicates</code> luettelee nämä parit olemassa olevasta tietokannasta sekä katkaistut ottelut ja niiden pidemmän version.</li>
<li><strong>Toisella kirjoitusasulla tunnettu nimi tallennetaan kanonisella nimellään.</strong> <em>Alias</em> kertoo, että ”Martin A.” on ”Alice Martinin” toinen kirjoitusasu (tai että tapahtuman nimi on toisen nimen toinen kirjoitusasu). Tuonnissa pelaajien ja tapahtuman nimet korvataan kanonisella nimellään, ja ottelu sijoitetaan kanonisen tapahtuman turnaukseen. Ottelun sormenjäljet säilyttävät tiedoston nimet: ennen aliaksen olemassaoloa tuotu tiedosto tunnistetaan siksi aina. Ottelu, jonka nopat ovat tietokannassa olevan ottelun nopat ja jonka nimet eroavat vain tunnettujen aliasten osalta, ei ole toinen ottelu: sen analyysit rikastavat tallennettua.</li>
</ul>
<p>Aliakset hallitaan asetusten <strong>Korpus</strong>-välilehdellä: valitse <strong>Pelaajat</strong> tai <strong>Tapahtumat</strong>, kirjoita alias ja kanoninen nimi tai poista alias luettelosta. <strong>Ehdota</strong> luettelee nimet, jotka eroavat vain kirjainkoon, aksenttien, välimerkkien tai sanajärjestyksen osalta; mitään ei oteta käyttöön ilman napsautusta <strong>Käytä</strong>. Samalla välilehdellä on valintaruutu <strong>Ohita kaksoiskappaleet tuonnissa</strong> (<code>--skip-duplicates</code>-valitsimen vastine, istunnon ajaksi) ja olemassa olevan tietokannan <strong>todennäköisten kaksoiskappaleiden</strong> haku, sama kuin <code>blunderdb repair --duplicates</code>: jokainen pari annetaan ottelunumeroillaan, mitään ei yhdistetä. Samoilla nopilla olevan parin alla <strong>Luo nämä aliakset</strong> tallentaa yhdellä napsautuksella pelaaja-aliakset, joiden jälkeen molemmat ottelut nimeäisivät samat pelaajat, ja uudemman ottelun kirjoitusasusta tulee alias: vain yksi ehdotus, kun nimi on yhteinen molemmille, muuten kaksi — paikka paikalta, sitten ristiin — joista valitaan se, joka kertoo, kuka on kuka. Komentoriviltä: <code>blunderdb players alias</code> ja <code>blunderdb events alias</code>.</p>
<ul>
<li><strong>Kansio tuodaan rinnakkain.</strong> Tiedostot luetaan usealla ytimellä yhtä aikaa ja kirjoitetaan ryhmittäin, aina kansion järjestyksessä: otteluiden numerot eivät riipu koneesta. Tiedostoa, joka on tavu tavulta sama kuin samasta kansiosta jo luettu tiedosto, ei lueta uudelleen: se lasketaan kaksoiskappaleeksi. Peruuttaminen pysäyttää tuonnin meneillään olevaan ryhmään; jo kirjoitettu säilyy.</li>
<li><strong>Edistyminen luetaan asemina sekunnissa.</strong> Tuontiikkuna näyttää luetun prosenttiosuuden, nopeuden, arvioidun jäljellä olevan ajan ja juoksevat määrät (tuodut, kaksoiskappaleet, virheelliset). <strong>Pienennä</strong> siirtää sen tilariville, josta merkki avaa sen uudelleen: tuonti jatkuu sillä aikaa kun työskentelet, ja ikkuna palaa lopuksi itsestään raportin kanssa. Sata ensimmäistä virhettä luetellaan; loput lasketaan, ja sovelluksen loki nimeää ne kaikki. Komentorivillä <code>blunderdb import --type batch</code> näyttää saman edistymisen virhetulosteessa, ja <code>--format json</code> palauttaa sen lopullisessa oliossa (<code>progress</code>).</li>
<li><strong>Jokainen tuonnin tiedosto kirjataan lokiin, ja keskeytynyt tuonti voidaan jatkaa.</strong> Erän loki säilyttää jokaisesta tiedostosta polun, koon, muokkauspäivämäärän, SHA-256-sormenjäljen ja tuloksen: uusi ottelu (numeroineen), kaksoiskappale (sen kattavan ottelun kera), rikastettu ottelu tai virhe (viestin kera). Peruttu tai katkennut tuonti jatkuu toistamatta jo päätettyä: tiedosto, jolla on sama polku, koko ja päivämäärä, ohitetaan lukematta; tiedosto, jolla on sama sisältö, luetaan mutta sitä ei analysoida; virheellinen tiedosto yritetään uudelleen. Sovelluksessa tuonnin päätösikkuna luettelee lokin (yksi tiedosto riviä kohti, tuloksineen ja virheen viestineen; uudelleen yritetty tiedosto näyttää viimeisen lopputuloksensa), ja ottelun tuottanut rivi avaa sen. <strong>Peruttu</strong> tuonti jättää ikkunan auki <strong>Jatka</strong>-painikkeen kanssa. Komentorivillä <code>blunderdb import --type batch --dir &lt;kansio&gt; --resume &lt;erä&gt;</code> jatkaa erää, jonka numero näytettiin tuonnin alussa, ja <code>--format json</code> palauttaa lokin lopullisessa oliossa (<code>journal</code>). Palvelin hyväksyy kentän <code>resume</code> pyynnössä <code>imports.batch</code> ja palauttaa lokin kutsulla <code>imports.files</code>. Loki on tuonnin tietoa: tietokannan avaaminen tai lukeminen ei kirjoita siihen mitään.</li>
<li><strong>Sovelluksen pysähtymisen katkaisema tuonti jatkuu ikkunasta.</strong> Komento <code>:resume</code> luettelee tuonnit, joita mikään ei ole päättänyt; valitse jatkettava ja osoita sitten kansio tai tiedostot uudelleen, kuten <code>--dir</code> komentoriviltä. Erän loki ratkaisee, mitä ei lueta uudelleen.</li>
<li><strong>Suuri kansio tuodaan massatilassa.</strong> 200 tiedostosta alkaen blunderDB kirjoittaa suuremmalla välimuistilla ja harvemmilla tarkistuspisteillä. Jos tietokannassa ei ole vielä yhtään asemaa, se menee pidemmälle: hakuindeksit rakennetaan uudelleen vasta lopuksi, eikä kirjoituksia enää synkronoida levylle. Sähkökatko tällaisen tuonnin aikana voi silloin vahingoittaa tietokantaa: se on luotava uudelleen ja tuonti käynnistettävä uudestaan. Ohjelman äkillinen pysähtyminen sen sijaan jättää vain puuttuvia indeksejä, jotka seuraava avaus rakentaa uudelleen (loki kertoo siitä).</li>
<li><strong>Mitä blunderDB ei koskaan kirjoita</strong>: uudelleen laskettu onni — se luetaan lähdetiedostosta tai jää tuntemattomaksi — ja rollout, jota se ei ole itse ajanut: <code>.xg</code>-tiedoston rollout-tietoja ei avata. Vain blunderDB:n itse tuottamat rollutit (Rolloutit) tallennetaan analyysin viereen.</li>
</ul>
<h3>Ottelupaneeli</h3>
<p><strong>Ottelupaneeli</strong> (<em>CTRL-Tab</em>) luettelee tuodut ottelut. Kaksoisnapsauta ottelua (tai paina <em>ENTER</em>) navigoidaksesi sen siirroissa. Komento <code>m</code> jatkaa navigointia viimeksi katsotussa ottelussa.</p>
<p>Paneelin yläreunan suodatuskenttä (<em>/</em> siirtyy siihen, <em>Esc</em> tyhjentää sen) säilyttää vain ottelut, joissa pelaaja, tapahtuma, paikka, turnaus tai päivämäärä sisältää kirjoitetun tekstin. Suodatuksen ja sarakkeiden lajittelun tekee tietokanta: luettelo latautuu sivuittain vierittäessä, ja laskuri ”n / N ottelua” näyttää ladatun osuuden. Pelaajan, päivämäärän tai turnauksen korjaaminen luettelossa päivittää vain muokatun rivin.</p>
<p>Kun luettelo on tyhjä, paneeli tarjoaa <strong>Tuo… (Ctrl+I)</strong>; ilman avattua tietokantaa se tarjoaa sen sijaan <strong>Avaa tietokanta…</strong> sekä <strong>Takaisin aloitusnäyttöön</strong>. Kun tekstisuodatin tyhjensi luettelon, se tarjoaa <strong>Tyhjennä suodatin</strong>. Tyhjät Stats-, Kokoelmat- ja Anki-paneelit tarjoavat samat painikkeet.</p>
<p>Käyttäjä voi:</p>
<ul>
<li>selata ottelun siirtoja näppäimillä <em>VASEN</em> ja <em>OIKEA</em>,</li>
<li>vaihtaa pelistä toiseen näppäimillä <em>PageUp</em> ja <em>PageDown</em>,</li>
<li>näyttää siirtojen analyysin (pelinappulat ja kuutio) painamalla <em>CTRL-L</em>,</li>
<li>vaihtaa pelinappulasiirtojen ja kuution analyysin välillä <em>d</em>-näppäimellä,</li>
<li>nähdä todella pelatun siirron korostettuna analyysissä,</li>
<li>hakea ottelun asemista komennolla <code>ss</code> (esim. <code>ss E&gt;80</code>); <em>Esc</em> palaa sen jälkeen tarkasteltuun siirtoon (katso Hakupaneeli).</li>
</ul>
<p>Kunkin ottelun viimeksi katsottu asema tallennetaan ja palautetaan automaattisesti. Näytä tai piilota paneeli painamalla <em>CTRL-Tab</em> tai suorittamalla komento <code>match</code>.</p>
<p>Rivin <strong>⊕</strong>-painike rikastaa ottelun tiedostosta. Sen takana ei ole mitään uutta: saman ottelun tuominen uudelleen toisessa muodossa rikastaa sen jo paikallaan — kanoninen tiiviste tunnistaa, että kyseessä on sama ottelu, ja toisen tiedoston analyysit ja kommentit täydentävät ensimmäistä. Painike tuo sen, että se löytyy: kukaan ei arvaa, että tuonti on myös rikastus. Seuraava raportti kertoo kumpi näistä tapahtui — ”rikastettu: 1” eikä ”tuotu: 1”.</p>
<p>Jokaisen ottelun voi viedä Jellyfish <code>.mat</code> -transkriptioksi otteluluettelon ⬇-painikkeella tai ottelun tietolomakkeen <em>.mat</em>-painikkeella.</p>
<p>Ottelun napsauttaminen avaa sen tietonäkymän. Sen välilehti <strong>Pelipöytäkirja</strong> luettelee siirrot peli kerrallaan, ja siirron napsauttaminen vie katselmoinnin siihen. Jokaisella siirrolla on siellä <strong>vakavuutensa</strong>: <code>?</code> virheelle, <code>??</code> blunderille, värillinen viiva rivin reunassa ja siirron hinta equityna, kun osoitin viedään merkinnän päälle. Rajat ovat tietokannan omat (Asetukset), samat joilla tilastot laskevat. Siirto arvioidaan sellaisena kuin se pelattiin: sama asema, joka pelattiin ottelussa kahdesti, saa kaksi arviota. Siirrolla, jota analyysi ei pisteytä, ei ole merkintää.</p>
<p>Jokaisen pelin otsikko laskee sen merkinnät, oli peli avattuna tai ei: näkee avaamatta, missä pelissä blunderit ovat.</p>
<p>Otteluvälilehden <strong>Infos</strong>-välilehti näyttää ottelun otsakkeen. Se lisää siihen, mitä lähdetiedosto kertoo pelaajista ja istunnosta, kun se kertoo — eXtreme Gammon -tiedosto kertoo aina: kunkin pelaajan Elo-luvun ja kokemuksen suluissa, litteroijan, rahapelin Jacoby- ja Beaver-säännöt sekä ohjelman, joka kirjoitti tiedoston. Nämä tiedot viedään ottelun mukana. Jo olemassa olevan tiedoston tuominen uudelleen antaa ne ottelulle, jolta ne puuttuivat, korvaamatta mitään, mitä ottelulla jo on. Komentorivin <code>match</code>-komento näyttää ne myös. eXtreme Gammon -tiedoston ottelun alku- ja loppukommenteista tulee ottelun kommentti, jonka allekirjoittaa tiedoston kirjaaja, tai <code>XG</code>, jos tiedosto ei nimeä kirjaajaa; kelloa ja equity-taulukkoa ei tuoda. Kortti näyttää tämän kirjoittajan kommentin vieressä, ja vienti kopioi sen mukana; kommentin muokkaus allekirjoittaa sen asetuksen <strong>Nimesi</strong> nimellä.</p>
<p>Paneelin työkalupalkin <strong>Yhdistä pelaajat</strong> -painike avaa ikkunan, joka luettelee kaikki tietokannan pelaajanimet otteluidensa määrän kanssa: valitse saman pelaajan eri kirjoitusasut, valitse säilytettävä kanoninen nimi ja yhdistä sitten. Yhdistäminen luo yhden aliaksen kutakin muunnelmaa kohden: ottelut säilyttävät tiedostojensa nimet, mutta tilastot, Pelaajat-taulukko ja haku <code>pl"…"</code> lukevat kaikki muunnelmat yhtenä pelaajana, ja seuraavat tuonnit tallentavat kanonisen nimen. Aliaksen poistaminen asetusten <strong>Korpus</strong>-välilehdellä kumoaa yhdistämisen.</p>
<p>Kun ottelu on avattu, laudan yläpuolelle ilmestyy <strong>tietopalkki</strong>: se muistuttaa läsnä olevista pelaajista (<em>pelaaja 1</em> vastaan <em>pelaaja 2</em>) sekä ottelun taustatiedoista (tapahtuma, paikka, kierros, päivämäärä ja ottelun pituus, kun nämä tiedot ovat saatavilla). Tämä palkki näytetään myös ottelutilan ulkopuolella: kun tutkittava asema (haun, kokoelman tai suoran haun tuloksena) on peräisin yhdestä tai useammasta ottelusta, palkki ilmoittaa sen <strong>alkuperän</strong> — ensimmäisen kyseessä olevan ottelun ja tarvittaessa « +N »-merkin, joka luettelee muut osoitettaessa. Erikseen tuotu asema, johon mikään ottelu ei viittaa, ei näytä mitään.</p>
<p><strong>Haku</strong>- ja <strong>Eval</strong>-välilehdet korvaavat laudan työlaudalla: laudan yläreunan nauha kertoo siitä (”Hakulauta”, ”Arviointilauta”), ja tietopalkki piilotetaan, kun se kuvaisi asemaa, jota ei ole näytöllä. Paluu analyysiin palauttaa tutkitun aseman.</p>
<p>Avattaessa otteluita sisältävää tietokantaa <strong>Ottelut</strong>-paneeli näytetään heti ja tarkastelu alkaa suoraan ensimmäisestä asemasta, jotta navigoinnin voi aloittaa välittömästi.</p>
<div class="admonition note">
<p>Tietokannan voi avata kirjoitustilassa vain yksi ikkuna kerrallaan. Jos avaat tietokannan, joka on jo avattu toisessa blunderDB-ikkunassa, se avautuu <strong>vain luku</strong> -tilassa: selaus, haku ja analyysi ovat edelleen mahdollisia, mutta kaikki muokkaus on poistettu käytöstä ja otsikkopalkissa lukee « [vain luku] ».</p>
</div>
<div class="admonition tip">
<p>Katso saatavilla olevat pikanäppäimet kohdasta Näppäimistöoikotiet.</p>
</div>
<h3>Litterointipaneeli</h3>
<p><strong>Litterointipaneeli</strong> (<em>CTRL-SHIFT-T</em>, komento <code>transcribe</code> tai <code>tr</code>) on tarkoitettu edessä olevan ottelun kirjoittamiseen — tuloslomakkeesta tai videotallenteesta — ja sen tekemiseen kirjaston otteluksi. Kirjoitettava on <strong>luonnos</strong>: se elää tietokannassa, sen voi sulkea ja avata uudelleen, eikä se tule mukaan tilastoihin eikä hakuihin ennen kuin se on tallennettu otteluksi.</p>
<p>Paneeli avautuu tietokannan <strong>luonnosluetteloon</strong>: viimeisin muutos, pelaajat, ottelun pituus, toimintojen määrä ja jo tuotettu ottelu (<code>#</code> ja sen tunniste) tai maininta ”ei ottelua”. Napsautus avaa luonnoksen, ja palkin <strong>Luonnokset</strong>-painike palaa luetteloon. <strong>Uusi transkriptio</strong> -painike avaa luontilomakkeen.</p>
<p>Luetteloa selataan myös näppäimistöllä: <em>ALAS</em> ja <em>YLÖS</em> (tai <em>j</em> ja <em>k</em>) siirtävät korostusta, <em>ENTER</em> avaa korostetun luonnoksen ja <em>n</em> avaa lomakkeen. Ensimmäinen luonnos on korostettuna avattaessa ja se on viimeksi muutettu: eilisen työn jatkaminen maksaa siis kaksi näppäintä, <em>CTRL-VAIHTO-T</em> ja sitten <em>ENTER</em>.</p>
<p>Lomake kysyy vain yhtä asiaa: <strong>ottelun pituutta</strong>. Arvo <code>0</code> tarkoittaa rahapeliä ja tuo näkyviin <em>Jacoby</em>- ja <em>Beaver</em>-ruudut. Kenttä avautuu viimeksi muokatun luonnoksen pituudella, tai arvolla 7, kun tietokannassa ei ole yhtään. Pelaajien nimiä ei kysytä: luonnos nimeää puolet <em>Pelaaja 1</em> ja <em>Pelaaja 2</em>, ja luettelo näyttää ”Nimetön”.</p>
<p>Kaikki, mikä johdetaan kirjoitetusta — ottelun pituus (tai ”Raha”), pistetilanne, maininta <em>Crawford</em> kun käynnissä oleva peli on sellainen, pelin numero, kuution tila — sen arvo, keskellä tai omistajansa nimissä — ja vuorossa oleva puoli — näkyy <strong>ottelupalkissa</strong> laudan yläpuolella: siellä katse jo on, kun kysyy kenen vuoro on. Odotettu toiminto sen sijaan kirjoitetaan kokonaisin sanoin <strong>tilapalkkiin</strong>: ”Kévinin nopat”, ”Alicen vastaus tuplaukseen”.</p>
<p><strong>Luonnospalkki</strong> paneelin yläreunassa sisältää vain eleet, jotka vievät luonnoksen ulos itsestään, kaksi kumoamisnuolta ja laudan suunnan.</p>
<p><strong>Pelaaja 1 pysyy laudan alalaidassa</strong> riippumatta siitä, kummalla on vuoro. Transkriboitava ottelu on käynnissä oleva peli: vuoro vaihtuu joka puolisiirrolla, ja sen seuraaminen kääntäisi laudan vuorosta toiseen — juuri katsotut nappulat siirtyisivät ylös ja silmä kulkisi saman matkan uudelleen joka heitolla. Vuoron näkee <strong>nopista</strong>, jotka vaihtavat puolta. Palkin ⇅-painike kääntää laudan ja näyttää pelaajan 2 alhaalla; se ei muuta luonnosta, ja näkymä palaa oikein päin, kun luonnos suljetaan. Ei pidä sekoittaa Metatiedot-paneelin <em>Vaihda pelaajat</em> -painikkeeseen, joka vaihtaa pelaajat itse asiakirjassa.</p>
<p>Palkin <strong>Metatiedot</strong>-painike avaa luonnoksen otsakkeen milloin tahansa: molempien pelaajien nimet — täydennettyinä kannan pelaajista —, tapahtuman, paikan, kierroksen, päivämäärän (oletuksena tämä päivä), kirjaajan (oletuksena kannan käyttäjä) ja turnauksen, johon ottelu liitetään tallennettaessa. Mikään kenttä ei ole pakollinen: nimetön luonnos tallentuu ja vie tiedostoon samoin, tyhjin otsakkein. <strong>Vaihda pelaajat keskenään</strong> -painike vaihtaa nimet keskenään, antaa kaikki toiminnot vastapuolelle ja kääntää laudan: se on sama ottelu, luettuna toiselta puolelta.</p>
<p><strong>Ottelun pituus</strong> vaihdetaan samasta paneelista, milloin tahansa: pisteet, Crawford-peli ja viitekehys — rahapelit kun pituus on <code>0</code>, jolloin <em>Jacoby</em>- ja <em>Beaver</em>-ruudut tulevat näkyviin — lasketaan uudelleen luonnoksen päästä päähän, ja voiton jälkeen kirjatut toiminnot merkitään «lopun jälkeisiksi» ilman että yhtäkään poistetaan. Pituus kuuluu aseman identiteettiin: tallennuksen jälkeen sen muuttaminen ja uusi tallennus kirjoittaa täysin uudet asemat, jotka on analysoitava, ja vanhat katoavat heti kun mikään ei enää pidä niitä.</p>
<p>Palkin alla luonnos jakautuu kolmeen alueeseen: hiirikohteiden <strong>paletti</strong>, <strong>siirtoehdokkaat</strong> ja <strong>transkriptio</strong>. Ne asettuvat paneelin leveyden mukaan. Leveässä paneelissa — alatelakassa — kaikki kolme ovat rinnakkain, paletti vasemmalla. Keskikokoisessa paneelissa ehdokkaat vievät yläosan ja transkriptio tulee heittokolmion viereen, siihen tilaan jonka kolmio jättää oikealle puolelleen. Kapeassa paneelissa kolme seuraavat toisiaan: ehdokkaat, paletti, transkriptio — kolmio ja transkriptio eivät mahdu siinä rinnakkain ilman että transkriptiolta katkeaa toinen sarake, ja telakan leventäminen muutamalla kymmenellä pikselillä riittää tuomaan ne yhteen. Joka tapauksessa sääntö on sama: mikään ei tule heiton kahden ruudun ja ensimmäisen ehdokasrivin väliin, ja vähintään viisi ehdokasta luetaan ilman mitään vierittämistä.</p>
<p>Paletti näyttää kaksi noppaa sitä mukaa kuin ne syötetään; napsautus niihin tyhjentää ne, kuten <em>ASKELPALAUTIN</em>. Peli alkaa ensimmäisellä siirrollaan, jonka pelaa aloitusheiton voittaja: sen kaksi noppaa kirjoitetaan kuten tuo heitto, pelaajan 1 noppa ja sitten pelaajan 2, ja suurempi antaa siirron puolelleen, joka pelaa molemmat nopat. Siirto valitaan sitten ehdokkaista kuten mikä tahansa muu. Tasapelejä, jotka heitetään pöydässä uudelleen, ei kirjata; tuplanoppana kirjoitettu ensimmäinen siirto tallennetaan sellaisenaan ja merkitään ”ristiriitaiset nopat”, koska mikään aloitusheitto ei ole tuplat, ja kuution tuplaus ennen ensimmäistä siirtoa merkitään ”mahdoton kuutiotoiminto”. Pelin päättymisen jälkeen kirjattu kuutiotoiminto saa saman merkinnän: se jää päättyneeseen peliin, ei avaa uutta ja poistetaan käsin.</p>
<p>Heti kun toinen noppa putoaa, luetellaan kaikki heiton <strong>lailliset siirrot</strong>, sisäisen moottorin järjestäminä, ensimmäinen esivalittuna ja sen nuolet laudalle asetettuina. Luettelo antaa siirron, sen ekviteetin ja eron parhaaseen: transkriptio on nähdyn siirron tunnistamista, ei sen arvostelua — siihen on <strong>Arviointi</strong>-paneeli. Tämä järjestys on arviointi: se näytetään, sitä ei koskaan kirjoiteta tietokantaan. Kun moottori ei ole käytettävissä, siirrot luetellaan järjestämättöminä ja luettelo kertoo sen otsikossaan.</p>
<p><strong>Rulla</strong> valitsee seuraavan tai edellisen ehdokkaan sekä luettelon että laudan päällä: katse pysyy laudalla ja nuolet vierivät ohi, mikä tunnistaa siirron nopeammin kuin sen notaation lukeminen. Napsautus riviin valitsee sen, kaksoisnapsautus vahvistaa sen.</p>
<p>Kahdenkymmenenyhden heiton kolmio on heiton kahden ruudun alla, näppäimistön vierellä eikä sen tilalla: kaksi numeroa on yhä kaksi kertaa nopeampi kuin napsautus, ja kolmio on sitä varten, joka kirjaa käsi hiirellä. Yksi ruutu heittoa kohti, ei koskaan kahta: 3-1 ja 1-3 ovat sama heitto.</p>
<p>Laudalla pelattu siirto säästää noppien lukemiselta. Niin kauan kuin yhtään noppaa ei ole syötetty, napsautus nappulaan ja sitten sen kohteeseen — tai veto toisesta toiseen — pelaa siirron laudalla laillisten siirtojen rajoissa; valitun nappulan tarjoamat kohteet syttyvät. Molemmat nopat päätellään askelista: 13/7 ja sitten 8/7 kertoo 6-1 ilman että numeroa on näppäilty, ja toiminto kirjataan heti kun siirto on valmis. Askelpalautin kumoaa viimeisen askeleen, numero hylkää siirron ja palaa noppien syöttöön, ja kaksoisnapsautus laudan ulkopuolella aloittaa sen alusta. Kun useampi heitto tuottaa saman siirron — ulosnosto, jonka useampi noppa kattaa, tai noppa jota ei voi pelata — mitään ei kirjata ja kolmio jättää napsautettaviksi vain nuo heitot: heittoa ei koskaan arvata pelin katsojan puolesta.</p>
<p>Kun molemmat nopat on syötetty, myös lauta pelaa, tämän heiton laillisten siirtojen rajoissa — asiakirjan lopussa kuten uudelleen tarkasteltavassa toiminnossa, jonka nopat kohdistin on ladannut. Jokainen pelattu askel jättää listaan vain sen sisältävät ehdokkaat, ensimmäinen niistä esivalittuna: tämä on kaukana listassa olevan siirron ele, kun alas kahdenteentoista ehdokkaaseen kulkeminen maksaa kolmetoista näppäilyä. Valmis laillinen siirto kirjataan heti, nopat sellaisina kuin ne näppäiltiin; uudelleen tarkasteltavassa toiminnossa se korvaa toiminnon.</p>
<p>Laiton siirto kirjataan sellaisena kuin se pelattiin, ilman painiketta ja ilman tilan vaihtoa. Kun nopat on syötetty, veto, jota mikään laillinen siirto ei tarjoa, laskee nappulan siihen, mihin se päästetään — myös pisteestä, josta ei lähde yhtään laillista siirtoa, kunhan siinä on vuorossa olevan puolen nappula. Siirto poistuu silloin säännöistä: loppu pelataan vapaasti, napsauttamalla tai vetämällä, ehdokaslista väistyy rivin tieltä, joka muistuttaa siitä, eikä mitään kirjata ennen ENTER-näppäintä, joka kirjoittaa syötetyt nopat, askeleet ja saadun laudan. Askelpalautin kumoaa viimeisen askeleen; ainoan sääntöjen vastaisen askeleen kumoaminen palauttaa listan. Ilman syötettyjä noppia veto pysyy rajoitettuna: laiton siirto ei kerro, mikä heitto sen tuotti.</p>
<p>Siirron voi myös kirjoittaa näppäimistöllä, transkriptissa. Kaksoisnapsautus siirron — tai tanssin, kirjaamattoman siirron — soluun muuttaa sen kentäksi, johon sen notaatio on valmiiksi täytetty. Siihen kirjoitetaan vain siirto, <code>13/7 8/7*</code>, <code>bar/22</code> tai <code>6/off</code>: nopat ovat solun omat. ENTER kirjaa sen kirjoitetun siirron tilalle, ESC sulkee solun kirjoittamatta mitään, ja teksti, joka ei ilmaise mitään siirtoa, jättää kentän auki. Kesken olevan syötön katkoviivasolu avautuu samoin, heti kun sen molemmat nopat on syötetty.</p>
<p>Vapaalla vedolla tai notaatiolla syötetty siirto, joka sattuu olemaan laillinen, pysyy tavallisena siirtona — vertailu tehdään saadusta laudasta, ei koskaan eleen alkuperästä; muutoin se merkitään transkriptiin ”laiton siirto”, ja <code>.mat</code>-vienti varoittaa ennen tiedoston kirjoittamista kieltäytymättä koskaan.</p>
<p>Noppien rivillä rivi <strong>Tuplaa</strong>, <strong>Ota</strong>, <strong>Passaa</strong>, <strong>Luovuta</strong> tuo hiirelle kuution neljä elettä: ne ovat kahden nopan kanssa viisi mahdollista vastausta yhteen ainoaan kysymykseen — mitä vuorossa oleva puoli teki? Se kertoo kenen vuoro on: vuorossa oleva puoli ilmoittaa — tuplaa, luovuttaa — tai vastapuoli vastaa — ottaa, passaa; ei koskaan kaikkia neljää yhtaikaa, ja painike, jonka ele ei vastaisi mihinkään, pysyy sammuneena. Näppäimistö sen sijaan ei koskaan kiellä mitään: sammunut painike on kohde, jota ei tarjota, ei kielletty ele. ”Luovuta” ei vielä kirjaa mitään: rivistä tulee kolme tasoa — yksinkertainen, gammon, backgammon — ja ”Peruuta”, joka kahdentaa ESC-näppäimen. Laudalle piirretty kuutio on näiden eleiden toinen kohde: napsautus siihen tarjoaa tuplausta. Tarjouksen edessä se ei vastaa — ottaminen ja passaaminen ovat kaksi symmetristä vastausta ja asuvat yhdessä rivillä, kumpikin yhden napsautuksen päässä.</p>
<p><strong>Tilapalkki</strong> kertoo yhdellä sanalla, mitä luonnos odottaa: itsestään kirjatun tanssin, pelin ensimmäisen siirron, tuplaukseen odotetun vastauksen, luovutuksen jälkeen odotetun tason, paikalla tehtävän korjauksen, siirron ”tarkistettavaksi”, jonka heitto on muuttunut. Se vastaa siellä myös eleille, joilla ei ole mitään tehtävää — ”ei mitään kumottavaa”, ”ei toimintoa kohdistimen alla” — puolentoista sekunnin ajan. Toiminnon jälkeensä jättämä epäjohdonmukaisuus sen sijaan ilmoitetaan transkriptin otsikossa, siellä missä virheellinen solu on.</p>
<p>Peli päättyy passiin, luovutukseen tai viidennentoista nappulan ulos kantamiseen (yksinkertainen, gammon tai backgammon, kerrottuna kuution arvolla). Pistetilanne, Crawford-peli ja ottelun loppu näkyvät silloin ottelupalkissa, ja seuraavan pelin ensimmäistä siirtoa odotetaan.</p>
<p>Pelin tilanne on se, jonka aiemmat pelit antavat, ellei pöydässä ilmoitettu toisin. Kaksoisnapsautus pelin otsikon tilanteeseen transkriptissa muuttaa sen valmiiksi täytetyksi kentäksi: siihen kirjoitetaan tilanne, jolla peli pelattiin — <code>3-2</code>, <code>3–2</code> tai <code>3 2</code> —, ENTER kirjaa sen, ESC sulkee kentän kirjoittamatta mitään, ja tyhjennetty ja sitten vahvistettu kenttä palaa johdettuun tilanteeseen. Peli pelataan tällä tilanteella: Crawford-peli, ottelun loppu ja seuraavat pelit seuraavat siitä, ja sekä tallennettu ottelu että <code>.mat</code>-tiedosto kantavat sen. Johdetusta poikkeava tilanne merkitään, työkaluvihje kertoo johdetun, ja pelin ensimmäinen toiminto kantaa epäjohdonmukaisuuden ”epäjohdonmukainen ilmoitettu tilanne”. Rahapelissä ei ole tilannetta ilmoitettavaksi.</p>
<p>Transkripti täyttää paneelin oikean puoliskon: yksi sarake pelaajaa kohti, yksi rivi vuoroa kohti, kuutiotoiminto ja pelin loppu toimijan sarakkeessa. Kohdistimen solu on kehystetty; kohdistimen siirtäminen tuo laudan takaisin kohdennetun toiminnon asemaan ja näyttää sen ehdokkaat kirjattu siirto valittuna. Epäjohdonmukaisuus (laiton siirto, kaksoisvuoro, mahdoton kuutiotoiminto, toiminto ottelun lopun jälkeen, ristiriitaiset nopat, kirjaamaton siirto, epäjohdonmukainen ilmoitettu tilanne) koristaa solunsa ja nimetään työkaluvihjeessä. Kirjaamaton siirto on uudelleen luetun <code>.mat</code>-tiedoston tapaus: gnubg kirjoittaa siihen <code>???</code>, kun se ei säilyttänyt pelattua siirtoa, heitto tunnetaan mutta siirtoa ei, ja kohdistimen asettaminen tähän soluun tarjoaa tämän heiton siirrot sen täydentämiseksi. Kaksoisvuoro jättää tyhjän, katkoviivalla kehystetyn solun sen puolen sarakkeeseen, jonka vuoro puuttuu: kohdistin pysähtyy siihen, napsautus vie sen sinne, ja siihen kirjoitetaan puuttuva vuoro — esimerkiksi poistettu päätös. Pelit taittuvat kokoon; kohdistimen peli on auki.</p>
<p><strong>Se mitä parhaillaan kirjoitetaan piirtyy transkriptiin</strong> katkoviivalla juuri siihen paikkaan, johon se kirjataan: nopat sitä mukaa kuin ne tulevat, siirron notaatio heti kun se on valittu, ja sen puolen sarakkeeseen, jolle toiminto kuuluu. Korjaus peittää solun, jonka se korvaa, lisäys avaa solun kahden naapurinsa väliin, ja uusi syöte ilmestyy käynnissä olevan pelin alaosaan. Mitään ei kirjoiteta luonnokseen ennen vahvistusta; se mitä luetaan ja se mitä asiakirja sanoo, eivät koskaan eroa toisistaan.</p>
<p>Korjaaminen on kirjoittamista siihen soluun, jossa ollaan. Kun kohdistin on toiminnon kohdalla, numero aloittaa sen heiton alusta paikallaan, ja myös neljä kuutiotoimintoa käyvät korjauksista: luovutuksen kohdalla <em>t</em> — tai <strong>Ota vastaan</strong> -painike — kirjoittaa oton luovutuksen <strong>tilalle</strong>, ilman että sitä täytyisi poistaa ja sitten lisätä. Peli jatkaa silloin kulkuaan: solu avautuu heti oton perään tuplaajan puolelle, ja pelin loput kirjoitetaan siihen tavalliseen tapaan, lisättynä seuraavan pelin ensimmäisen siirron eteen, kunnes peli päättyy. Samat näppäimet täyttävät solun, jonka lisäys on juuri avannut: <em>i</em> ja sitten <em>d</em> lisää tuplauksen kohdennetun toiminnon eteen. Tämä ensimmäinen siirto säilyttää tilanteen, josta sen peli alkoi, nyt ilmoitettuna: jos jatketun pelin loppu antaa toisen, ero merkitään.</p>
<p>Palaaminen <strong>viimeiseen</strong> toimintoon on paluuta sinne, missä litterointia kirjoitetaan. Sen kohdalla kirjoitettu numero korjaa yhä sen heittoa, mutta kun heitto on kirjoitettu uudelleen, seuraava numero vahvistaa toiminnon ja avaa seuraavan päätöksen; ENTER vahvistaa sen samoin. Seuraavat päätökset lisätään silloin perään, kuten ensimmäisellä kerralla.</p>
<p><strong>Toisen heiton</strong> kirjoittaminen pelin <strong>ensimmäiseen siirtoon</strong> ratkaisee uudelleen, kuka aloittaa: pelaajan 1 noppa kirjoitetaan ensin, pelaajan 2 sen jälkeen, ja suurempi voittaa — iso noppa ensin, siirto kuuluu pelaajalle 1, laudan alaosassa; pieni noppa ensin, pelaajalle 2, yläosassa. Uuden heiton ensimmäinen ehdokas esivalitaan ja siirto on tarkistettava. <strong>Saman heiton</strong> kirjoittaminen uudelleen, kummassa järjestyksessä tahansa, ei muuta mitään. Ensimmäinen siirto annetaan toiselle pelaajalle heittoa muuttamatta näppäimellä <em>s</em>: noppien järjestys seuraa puolta, ja ensimmäinen siirto, jonka tekee puoli, jota järjestys ei osoita, merkitään ”epäjohdonmukaiset nopat”. Pelin loppu säilyttää puolensa.</p>
<p>Keskelle asiakirjaa tehty <strong>lisäys jatkaa lisäämistä</strong>: vahvistus avaa tyhjän solun sen perään, ja seuraava toiminto lisätään vuorostaan sen sijaan että se korvaisi jälkimmäisen. Juuri tämä sallii kirjata jälkikäteen kokonaisen pelin lopun — luovutuksen, jonka olisi pitänyt olla otto — menettämättä sitä, mitä seuraavasta pelistä on jo kirjoitettu. Pelin loppu tai kohdistimen siirtäminen päättää lisäyksen: kohdistin asettuu silloin seuraavan pelin ensimmäisen siirron kohdalle.</p>
<p><strong>Del</strong> (tai <em>x</em>) poistaa muokattavana olevan päätöksen ja palaa edelliseen, valmiina korjattavaksi: kirjoitetussa solussa toiminto katoaa; avoimessa lisäyksessä tai asiakirjan loppuun kirjoitetussa heitossa hylätään syöttö. Toistuvasti painettu Del kulkee näin transkriptia taaksepäin pyyhkien. Seuraavat toiminnot säilyttävät puolensa, ja poiston jättämä kaksoisvuoro merkitään viemättä kohdistinta sen kohdalle.</p>
<p>Napsautus hiiren oikealla painikkeella soluun avaa tämän toiminnon korjaukset — lisää ennen, lisää jälkeen, poista, vaihda puolta — ja tuo kohdistimen sen päälle matkalla; ne ovat samat eleet kuin näppäimet <em>i</em>, <em>a</em>, <em>x</em> ja <em>s</em>, ja selaimen valikko poistetaan vain siellä. Muualla niillä ei ole painikkeita: painike, joka vaikuttaisi ”kohdistimen alla olevaan toimintoon”, tähtäisi soluun, jota ei ehkä näe, kun taas oikea napsautus nimeää omansa.</p>
<p>Luonnospalkki sisältää sen kaksi ainoaa poistumistietä. ”<strong>Valmis</strong>” (CTRL-ENTER) kirjoittaa ottelun kirjastoon ja vapauttaa luonnoksen; vain uusien asemien analyysi alkaa heti, edistymisineen ja peruutuksineen tilapalkissa. ”<strong>Hylkää</strong>” poistaa luonnoksen ilman ottelua; vahvistus kysytään vain luonnokselta, jota ei ole koskaan viimeistelty, koska se vie mukanaan kaiken siihen kirjoitetun. Vieressä palkki kertoo, mitä Valmis tekee — uuden ottelun tai ottelun #<em>n</em> korvaamisen. Se ei sano mitään luonnoksen omasta tallennuksesta: se kirjoitetaan tietokantaan jokaisen toiminnon jälkeen, ja paluu luetteloon jättää sen jatkettavaksi myöhemmin.</p>
<p>”<strong>.mat-teksti</strong>” avaa Jellyfish-tiedoston sellaisena kuin se kirjoitettaisiin, ikkunassa, joka on riittävän leveä pitämään sen sarakkeet kohdakkain, ja painikkeella sen kopioimiseen. ”<strong>Vie .mat</strong>” kirjoittaa saman tiedoston levylle. Kaksi nuolta <strong>↶</strong> ja <strong>↷</strong> kumoavat ja tekevät uudelleen, kuten <em>CTRL-Z</em> ja <em>CTRL-VAIHTO-Z</em>.</p>
<p>Jos litteroidun ottelun analyysi keskeytyi — sovellus suljettiin erän ollessa kesken —, tilarivi kertoo siitä seuraavalla kerralla, kun tietokanta avataan, ja tarjoutuu viemään sen loppuun. Keskeytyksestä ei jää mitään muistiin: tarjous palaa niin kauan kuin asemia on analysoimatta, ja uudelleen käynnistetty erä koskee vain tätä ottelua, ei koskaan koko kirjastoa.</p>
<p>Epäjohdonmukaisuuksia sisältävä luonnos viimeistellään silti varoituksen jälkeen: mitään ei hylätä. Laiton siirto viedään sellaisena kuin se pelattiin, varoituksella, että gnubg ja XG ilmoittavat siitä (”Invalid move”) ja poikkeavat sen jälkeen.</p>
<p><strong>Ottelupaneeli</strong> muistuttaa jokaisesta kesken olevasta luonnoksesta otteluluettelon yläpuolella: rivi ”Luonnos kesken” avaa Litterointi-välilehden.</p>
<p>Kirjaston ottelun korjaamiseksi ottelulistan ⌨-painike tai sen kortin ”<strong>Muokkaa litterointia</strong>” avaa luonnoksen kyseisestä ottelusta — tai avaa uudelleen sen, joka on jo auki — yksi luonnos ottelua kohden. Luonnoksen viimeistely korvaa ottelun samalla tunnisteella; muuttumattomien toimintojen asemat säilyttävät kommenttinsa, analyysinsä ja korttinsa. Tuotu ottelu (XG, GnuBG, BGF) sisältää analyysejä ja kommentteja, joita <code>.mat</code> ei sisällä: ennen avaamista valintaikkuna kertoo, kuinka monta niitä on enintään, ja että luonnoksen viimeistely voi menettää ne.</p>
<div class="admonition tip">
<p>Katso saatavilla olevat pikanäppäimet kohdasta Näppäimistöoikotiet.</p>
</div>
<h3>Turnauspaneeli</h3>
<p><strong>Turnauspaneeli</strong> (<em>CTRL-Y</em>) mahdollistaa otteluiden ryhmittelyn turnauksiin järjestelmällistä seurantaa ja tapahtumakohtaista tilastollista analyysiä varten. Turnauksia voi luoda, nimetä uudelleen ja poistaa; otteluita voi liittää niihin. Stats-paneelin tilastoja voi suodattaa turnauksen mukaan. Näytä tai piilota paneeli painamalla <em>CTRL-Y</em>.</p>
<p><strong>Uusi turnaus</strong> avaa luontikentän, joka saa kohdistuksen; <em>ESC</em> tai <strong>Peruuta</strong> sulkee sen. Napsautus korostaa rivin, kaksoisnapsautus tai <em>ENTER</em> avaa turnauksen: sen muistiinpanot, sitten sen ottelut, yksi per rivi, jotka kaksoisnapsautus avaa, ▲ ja ▼ järjestävät uudelleen, ⇄ vaihtaa pelaajia ja × poistaa turnauksesta. Kenttä <strong>Lisää ottelu…</strong> sijoittaa siihen tietokannan ottelun, ja ← palauttaa turnausluetteloon.</p>
<p>Turnaukset täyttyvät itsestään tuonnin yhteydessä. XG-, GnuBG- ja BGF-tiedostot nimeävät tapahtumansa; kun uusi ottelu tuodaan, blunderDB sijoittaa sen tämännimiseen turnaukseen ja luo turnauksen, jos sitä ei vielä ole. Turnauksen päivämäärä ja paikka jäävät tyhjiksi — ne täytetään täällä. Tietokannassa jo olevaa ottelua ei koskaan siirretä: sen tiedoston tuominen uudelleen ei kumoa käsin tehtyä järjestelyä.</p>
<p>Kunkin turnauksen <strong>PR</strong>- ja <strong>MWC</strong>-sarakkeet näyttävät <strong>viitepelaajan</strong> PR-arvon ja MWC-menetyksen — eli sen pelaajan, joka esiintyy turnauksen useimmissa otteluissa (tasatilanteessa se, joka teki eniten päätöksiä). PR ei siis sekoita omaa peliäsi vastustajiesi peliin: omissa turnauksissasi se kuvastaa yksin sinun suoritustasi. Viitepelaajan nimi näkyy työkaluvihjeenä, kun viet osoittimen arvon päälle.</p>
<h3>Turnauksen johtaminen</h3>
<p>blunderDB osaa <strong>johtaa</strong> turnauksen, ei vain arkistoida sitä. Johtamisen kantaa <strong>Nicomaque</strong>-moottori, tekijänään Nicolas Harmand: se pitää muodon, paritukset, kaaviot ja sijoitukset; blunderDB antaa sille käyttöliittymän ja säilyttää sen ottelut. Johtamisnäkymän otsikon <strong>ⓘ</strong>-painike muistuttaa tästä ja johtaa moottorin repositorioon ja dokumentaatioon.</p>
<p>Johdettava turnaus valitaan Turnauspaneeli -paneelista (<em>CTRL-Y</em>, komento <code>direct</code>): avaa turnaus ja valitse <strong>Johda tätä turnausta</strong>. Jo johdettu turnaus näyttää tilansa nimensä vieressä ja painike muuttuu muotoon <strong>Avaa johtaminen</strong>. Niin kauan kuin johtaminen on auki, pääalue näyttää turnauksen <strong>laudan sijasta</strong> — blunderDB:n ainoa poikkeus tästä säännöstä; mille tahansa muulle välilehdelle siirtyminen tuo laudan takaisin. <strong>Poistu johtamisesta</strong> näkymän otsikossa tai <strong>Sulje johtaminen</strong> paneelissa sulkee sen.</p>
<p>Johtamisella on kolme tilaa: <strong>valmistelussa</strong>, kunnes yhtään ottelua ei ole aloitettu, sitten <strong>käynnissä</strong>, ja <strong>päättynyt</strong>, kun turnaus on suljettu ja sijoitukset lukittu. Suljetun turnauksen voi avata uudelleen, ja se kysyy vahvistusta: loppusijoitus lakkaa olemasta lopullinen.</p>
<p>Kaikki päätetty kirjoitetaan <strong>lokiin</strong>, eikä mitään muuta kirjoiteta. Sijoitukset, kaaviot, ehdotukset ja varoitukset toistetaan tästä lokista joka avauksella: sähkökatko ei maksa mitään, eikä korjaus koskaan pyyhi tapahtunutta — se lisätään siihen.</p>
<h4>Johtamissivu</h4>
<p>Täällä johtaja viettää suurimman osan ajastaan. Ylhäältä alas: moottorin <strong>varoitukset</strong>, jotka pysyvät näkyvissä eivätkä koskaan estä mitään; <strong>pöytäruudukko</strong>, jonka otsikkorivillä on <strong>Tulosta lomake</strong> -painike parituksille; <strong>viimeisin päätös</strong>; <strong>ehdotuslista</strong>; ja vapaat pelaajat. Ruudukko tulee ennen listaa: pitkä lista ei koskaan työnnä sitä pois näytöltä. Näppäimistöllä sivun ensimmäinen <em>TAB</em>-pysäkki on ”Siirry jonoon”, joka siirtää kohdistuksen jonoon ruudukon läpi kulkematta; jono muistuttaa otsikkonsa alla pikanäppäimistään (<em>J</em> ja <em>K</em>, <em>ENTER</em>, <em>VASEN</em> ja <em>OIKEA</em>).</p>
<p>Näkymä täyttää pääalueen koko leveyden, ja jokainen välilehti vierittyy erikseen: kun välilehdestä poistuu ja palaa, tai siirtyy kilpailusta toiseen, paikka on se, johon sen jätti. Painikkeet ja kentät ovat vähintään 44 pikseliä korkeita, jotta niihin osuu tarkkuutta vaatimatta tiskin ääressä; ruudukon sarakkeiden määrä seuraa alueen leveyttä, ei ikkunan. <strong>Asetuksissa</strong> jokainen osio taittuu otsikkoonsa, ja Asetusten <strong>Avaa selaimessa</strong> -painike sekä otsikkorivin <strong>Seinäsivu</strong>-painike avaavat seinäsivun yhdellä napsautuksella, kun tulostuskansio on valittu; tilarivi näyttää kirjoitetun tiedoston polun. Ilmoittautuminen, myöhästynyt, seuraava vaihe, arvonta, ottelun aloitus ja päättäminen jättävät myös sinne palautteen (”Sophie Martin ilmoittautui — 16 ilmoittautunutta”).</p>
<p>Turnauksen johto avautuu <strong>Pelaajiin</strong>, kunnes turnaus on aloitettu, sen jälkeen <strong>Johto</strong>-välilehdelle, ja uudelleen avattaessa välilehdelle, jolle se jäi. Sovelluksen uudelleenlatauksen tai uudelleenkäynnistyksen jälkeen viimeksi avoinna ollut johto avautuu itsestään samalle välilehdelle. Toinen napsautus kohtaan <em>Johda</em> tai <em>Avaa johto</em> näkymän latautuessa ohitetaan.</p>
<p>Ehdotetun kierroksen voi ilmoittaa ennen sen käynnistämistä: <strong>Tuleva kierros…</strong> <em>Tulosta lomake</em> -painikkeen vieressä kysyy tulostettavan päivän ja kellonajan (”maanantai 21.9. klo 20”) ja tulostaa jonon parien arkin merkittynä ”ilmoitetuksi”. Mitään ei käynnistetä eikä kirjoiteta lokiin: kierros käynnistetään sinä päivänä, aikanaan. Vapaata pöytää odottavalla parilla on pöytänumeron paikalla viiva.</p>
<p>Turnausnäkymän yläreunassa <strong>kellopalkki</strong> mahtuu yhdelle riville: kellonaika, ensimmäisestä käynnistetystä ottelusta kulunut aika, pelatut ja käynnissä olevat ottelut, havaittu tahti minuutteina pistettä kohden suunniteltuun verrattuna, hitaat ottelut, seuraava tauko ja <strong>arvioitu loppu</strong>. Arvioitu loppu on moottorin ennuste: se toistaa lokin, pelaa turnauksen loppuun viisitoista kertaa suunnitellulla tahdilla, ja palkki näyttää mediaanin ilmoitettujen taukojen jälkeen siirrettynä. Yö, jota ei ole ilmoitettu tauoksi, lasketaan siis peliajaksi. Kellonaika, joka ei ole tältä päivältä, näyttää päivänsä.</p>
<p>Toisesta päivästä alkaen kulunut aika vaihtuu <strong>pelipäiväksi</strong> (ensimmäisen käynnistetyn ottelun päivä on päivä 1) ja <strong>peliajaksi</strong>: ajaksi, jolloin vähintään yksi ottelu oli käynnissä, ilman öitä ja taukoja, joina yksikään pöytä ei pelannut. Päätetyllä turnauksella ei ole enää kellopalkkia.</p>
<p>Ehdotus vahvistetaan <strong>yhdellä napsautuksella</strong> <em>Aloita</em>-painikkeesta. <strong>Aloita kaikki</strong> vahvistaa kahdella napsautuksella ne ehdotukset, joilla on pöytä, näytettyään ensin niiden listan; <em>Vahvista</em> on tuon listan alussa. Parit ilman pöytää jäävät listaan merkinnällä ”ei vapaata pöytää”: kierrostilassa kierros pysyy auki, kunnes kaikki sen pelaajat ovat mukana. Varasija tasatilanteessa olevien kesken odottaa myös johtajan valintaa, ja vaiheen vaihto sen mukana. ”Ohita toistaiseksi” ei kirjoita mitään: moottori on deterministinen, ja ehdotus palaa samanlaisena seuraavalla kerralla. <em>Muodosta pari käsin</em> on aina tarjolla — moottori ehdottaa, johtaja päättää.</p>
<p>Käsin paritettu ottelu ilman pöytänumeroa saa ensimmäisen vapaan pöydän; pöytä, jolla on ottelu käynnissä, hylätään. Jos kaikki pöydät on varattu, ottelu käynnistetään silti, ja sen ruutu näkyy ruudukon lopussa merkinnällä ”ei pöytää”, kunnes se siirretään pöytään.</p>
<p>Ehdotus voi kantaa moottorin huomautuksen: vapaata pöytää ei ole, tai ottelun odotettu päättyminen osuisi tauolle. Se on aloitettavissa kummassakin tapauksessa. Kun vaihe toimii <strong>mikrokierroksin</strong>, lista näyttää jäljellä olevan ajan seuraavaan erään; määräajan tullen ehdotukset ilmestyvät itsestään, eikä mikään käynnisty itsekseen.</p>
<h4>Tuloskortti</h4>
<p>Napsautus varattuun pöytään avaa ottelun kortin. Siinä on kaksi suurta kohdetta: <strong>kummankin pelaajan nimi</strong>. Voittajan napsauttaminen kirjaa tuloksen — kaksi napsautusta kaikkiaan, pöytä mukaan lukien. Voittaja on ainoa vaadittu tieto; tulos on vapaa, toinen, molemmat tai ei kumpaakaan. Näppäimistöllä <em>VASEN</em> tai <em>OIKEA</em> valitsee voittajan ja <em>ENTER</em> kirjaa hänet. Kortti sulkeutuu vasta, kun tulos on kirjoitettu: virhe jättää sen auki viestineen.</p>
<p>Kortin <strong>⋯</strong>-painike avaa harvoin tarvittavat toiminnot: luovutuksen (forfait) — jokainen painike nimeää poissaolevan ja voittajan —, vapaan huomautuksen (”aika loppui”, ”keskeytetty syystä…”), ottelun siirron toiselle pöydälle ja sen mitätöinnin. Luovutus ja mitätöinti vahvistetaan; luovutus <strong>Ilmoita luovutus</strong> -painikkeella, joka tarjoaa samalla kortilla myös häviäjän poistamista. Varattuun pöytään siirrettynä ottelu vaihtaa pöytänsä pöydän haltijan kanssa: kaksi ottelua ei koskaan jaa pöytää, ja sama ele palauttaa ne paikoilleen. Jos vanha loki on jättänyt kaksi samalle pöydälle, ruudukko näyttää molemmat ruudut merkittyinä, kunnes toinen siirretään.</p>
<p>Heti huomattu kirjausvirhe perutaan kahdella napsautuksella ruudukon alta: <strong>Korjaa</strong> viimeisin päätös, sitten oikea voittaja (<em>CTRL-Z</em> avaa saman peruutuksen). Vanhempi korjaus tehdään historiasta.</p>
<h4>Kontekstivalikot</h4>
<p>Napsautus hiiren oikealla, <em>MENU</em>-näppäin tai <em>SHIFT-F10</em> Direction-sivun kohteella avaa sen tavalliset toiminnot ilman kortin kautta kiertämistä: ruudukon ruutu (vapaa tai varattu), pelaaja (<strong>Pelaajat</strong>-välilehti, vapaat pelaajat), kaavion paikka, paikka, jonon ehdotus, historian rivi. Valikko avautuu kohteen kohdalle; <em>YLÖS</em> ja <em>ALAS</em> liikkuvat siinä, <em>ENTER</em> valitsee, <em>ESC</em> sulkee sen ja palauttaa kohdistuksen kohteeseen.</p>
<ul>
<li>Varattu ruutu: kirjaa tulos, kumman tahansa luovutus, vaihda pöytää (varattuun pöytään osoittaminen vaihtaa kaksi ottelua keskenään), peruuta ottelu, kunkin pelaajan historia.</li>
<li>Vapaa ruutu: aloita tässä valittu pariutus, poista pöytä käytöstä tai palauta se käyttöön (tapahtuman pöydät). Toiselle kilpailulle varattu pöytä ei tarjoa mitään.</li>
<li>Pelaaja: kirjaa käynnissä olevan ottelun tulos, siirry hänen pöytäänsä, historia, pariuta käsin toisen vapaan pelaajan kanssa, siirry toiseen kilpailuun, jossa hän myös pelaa, merkitse poissaolevaksi tai paikalla olevaksi, vetäydy nyt tai ottelun jälkeen, ilmoita uudelleen, korjaa tiedot.</li>
<li>Paikka ilman ottelua: liitä tuotu ottelu, jota tämä paikka odottaa.</li>
<li>Pariutus: aloita, aloita pöydässä…, vaihda pituutta…, pari toisin (nämä kolme valintaa avaavat käsin pariuttamisen kahden pelaajan, pariutuksen pituuden ja pöydän kanssa), ohita toistaiseksi, tulosta kierroksen lehti.</li>
<li>Historian rivi: korjaa tai mitätöi, lisää huomautus, suodata toisen pelaajan mukaan.</li>
</ul>
<p>Luovutus, ottelun peruutus ja pelaajan vetäytyminen säilyttävät vahvistuksen, joka niillä on kortissa ja rivien painikkeissa. Kun toiminto on käynnissä, toimivat valinnat näkyvät harmaina kuten painikkeet. Vain yksi valikko on auki kerrallaan: toisen avaaminen sulkee ensimmäisen.</p>
<p>Näppäimistöllä ruudukko vie vain yhden <em>TAB</em>-pysähdyksen: jokainen ruutu, myös vapaa, saa kohdistuksen, ja <em>VASEN</em>, <em>OIKEA</em>, <em>YLÖS</em>, <em>ALAS</em>, <em>HOME</em> ja <em>END</em> siirtävät ruudusta toiseen. Numero avaa kyseisen numeron pöydän kortin; yli 9:n pöydässä toinen numero kirjoitetaan 0,4 sekunnin kuluessa. <em>M</em> avaa kortin pöytäkentässä, ja myös <em>X</em>, olipa kortti jo auki tai ei: varattuun pöytään osoittaminen vaihtaa kaksi ottelua keskenään.</p>
<p>Hiirellä varattu ruutu <strong>vedetään</strong> toisen päälle: vapaaseen ruutuun ottelu vaihtaa pöytää; varattuun ruutuun rivi ”Pöytä 3 ↔ Pöytä 7?” pyytää vahvistamaan kahden ottelun <strong>vaihdon</strong>. Haamukuva seuraa osoitinta ja kohderuutu korostuu; <em>ESC</em> peruu eleen, eikä mitään kirjoiteta ennen kuin osoitin vapautetaan ruudun päällä. Käytöstä poistettu pöytä hylätään, ja tilarivi kertoo syyn. Komentorivillä <code>blunderdb tournament move</code> tekee saman siirron tai vaihdon (katso Komentoriviliittymä (CLI)).</p>
<h4>Koko näyttö</h4>
<p>Turnauksenjohdon sivun <em>F11</em>-näppäin tai oikean alakulman painike asettaa sivun koko näytön tilaan: työkalupalkki, välilehdet, lauta, paneeli ja tilapalkki katoavat ja koko ikkuna annetaan turnauksenjohdolle. Tila säilyy johdon välilehtien (Johto, Pelaajat, Historia, …) välillä. Avoin valikko tai kortti sulkeutuu ensin näppäimellä <em>ESC</em>; toinen <em>ESC</em> tai <em>F11</em> poistuu koko näytöstä ja palauttaa ikkunan aiempaan tilaansa. Tila päättyy myös sivulta poistuttaessa, vaihtamalla sovelluksen välilehteä tai sulkemalla johdon.</p>
<h4>Pikahaku</h4>
<p>Direction-sivun <em>/</em>-näppäin avaa paletin pelkästään turnaukselle: avoimen tapahtuman pelaajat, pöydät, käynnissä olevat ottelut ja kilpailut, tai pelkän kilpailun, kun se ei ole missään tapahtumassa. Kirjoitetaan nimi, seura tai pöydän numero (”4” tai ”t4”); pöydässä olevat pelaajat tulevat ennen vapaita pelaajia. <em>ENTER</em> vie kohteeseen: pelaaja ottelussa, ottelu tai varattu pöytä avaavat pöydän kortin, vapaa pöytä saa kohdistuksen ruudukossa, vapaa pelaaja näytetään <strong>Pelaajat</strong>-välilehdellä nimellään suodatettuna, kilpailusta tulee nykyinen välilehti. Tapahtuman toisen kilpailun tulos vaihtaa ensin kilpailua. <em>ESC</em> sulkee avaamatta mitään, ja syöttökentässä näppäin pysyy vinoviivana. <em>CTRL-SHIFT-P</em> avaa täyden paletin, jossa turnaus on myös mukana.</p>
<h4>Pelaajat</h4>
<p><strong>Pelaajat</strong>-välilehti ilmoittaa, korjaa ja poistaa. Ilmoittautumiskenttä säilyttää kohdistuksen ja tyhjenee jokaisen nimen jälkeen: kaksikymmentä pelaajaa ilmoitetaan pelkällä näppäimistöllä. Täydennys tarjoaa tietokannan pelaajia; yhden valitseminen lukitsee sen tarkan kirjoitusasun, jonka hänen ottelunsa kantavat, ja esitäyttää luokituksen hänen PR:llään.</p>
<p><strong>Hakemisto</strong> kokoaa kaikkien tietokannan johdettujen turnausten osallistujat, nimen mukaan yhdistettyinä, viimeisimmän ilmoittautumisen seuran ja luokituksen kera. Sitä ei koskaan tallenneta: johtamisen poistaminen poistaa siitä sen osallistujat. Aiemman turnauksen osallistujien ottaminen on yksi napsautus, olipa heitä kuinka monta tahansa; hakemisto kopioidaan CSV:nä tai tallennetaan tiedostoon (<strong>Tallenna…</strong>), ja se luetaan takaisin liitettynä.</p>
<p>Nelinpeliturnaus ilmoittaa pareja: ruutu <strong>Pari</strong> lisää parin toisen nimen, seuran ja luokituksen. Pari pelaa nimellä ”A / B”, jota myös sen ottelut kantavat; sen luokitus on kahden keskiarvo, ja kenttään <strong>Parin luokitus</strong> syötetty arvo korvaa sen. Hakemisto säilyttää kaksi henkilöä, ei koskaan paria.</p>
<p>Ennen <strong>Ilmoita</strong>-painiketta liitetyn CSV:n esikatselu luettelee lukukelvottomat rivit – ei nimeä, ei erotinta kun muilla riveillä on, luokitus joka ei ole luku – sekä kaksoiskappaleet, liitetyn sisällä tai jo ilmoitetun pelaajan kanssa. Kaksoiskappaletta ei ilmoiteta, ellei sen ruutua valita.</p>
<p>Luettelon suodatuskenttä säilyttää tekstinsä, kun kilpailua vaihdetaan: niin kauan kuin se on käytössä, sen viereen tulee <strong>suodatin: …</strong> -merkki, jonka rasti tyhjentää sen.</p>
<p>Arvonnan jälkeen saapuva <strong>myöhästyjä</strong> ottaa vapaan vapaakierroksen, jos kaaviossa on sellainen, ja käyttöliittymä kirjoittaa kentän viereen, mihin hän tulee mukaan, ennen vahvistusta. Jos vapaata paikkaa ei ole, hänet ilmoitetaan silti ja näkymä kertoo, mihin vaiheeseen hän tulee. Jo tehtyä arvontaa ei koskaan tehdä uudelleen.</p>
<p>Poistuminen tapahtuu <em>heti</em> tai <em>käynnissä olevan ottelun jälkeen</em>, sen mukaan lähteekö pelaaja saman tien vai pelaako loppuun; se vahvistetaan <strong>Poista</strong>-painikkeella, joka ei ole punainen, koska mitään ei poisteta.</p>
<p>Pelaajaa, joka jää pois yhdestä kierroksesta, ei tarvitse vetää pois: <strong>Merkitse poissaolevaksi</strong> hänen rivillään avaa pienen lomakkeen nimen alle — <em>kunnes</em> tiettyyn kellonaikaan (esitäytetty seuraavaan tuntiin), tai, kun kuluva vaihe on kierroksittainen sveitsiläinen, <em>kierrokseen asti</em> sen numerolla. Moottori lakkaa tällöin vain parittamasta häntä, mutta hänen sijansa, elämänsä ja paikkansa taulukossa pysyvät niinä, jotka hän on ansainnut — poissaolo ei ole luovutus. <strong>Palaa</strong> hänen rivillään poistaa poissaolon yhdellä klikkauksella, ennen ilmoitettua määräaikaa tai sen jälkeen.</p>
<p>Vetäytyneen pelaajan tietojen korjaaminen — nimi, seura, luokitus — jättää hänet vetäytyneeksi. Paluu on oma toimintonsa: <strong>Palauta</strong> hänen rivillään. Hänet paritetaan taas, niillä tuloksilla ja elämillä jotka hänellä oli lähtiessään; vetäytymisessä luovutuksella hävityt ottelut pysyvät hävittyinä.</p>
<p>Ryhmästä jatkoon päässyt, joka vetäytyy ennen seuraavan vaiheen arvontaa, jättää paikan vapaaksi. Lista ehdottaa silloin <strong>varasijaa</strong>: hänen ryhmänsä seuraava, joka ei ole jatkoon päässyt eikä vetäytynyt ja jolla on eniten ryhmävoittoja, ottaa hänen paikkansa. Tasatilanteessa lista ehdottaa jokaista heistä ja johtaja valitsee; ”Seuraava vaihe” ilman varasijaa jättää paikan vapaakierrokseksi. Kun kaavio on arvottu, vetäytynyt häviää ottelunsa luovutuksella.</p>
<h4>Kaaviot, paikat, sijoitukset, historia</h4>
<p><strong>Kaaviot</strong>-välilehti piirtää pudotuspelikaaviot yhdistäviin viivoineen ensimmäiseltä kierrokselta finaaliin, lohdutuskaavion pääkaavion viereen ja sveitsiläisessä järjestelmässä elämätaulukon. Alkulohko luetaan tulosten ristiintaulukkona. Vielä arpomaton kaavio näyttää harmaan runkonsa. Jo pelattu ottelu näyttää tuloksensa; moottorin ilmoittama ottelu merkitään paikalleen. Välilehden nimen vieressä oleva piste osoittaa, että kaavio on käynnissä.</p>
<p><strong>Paikan napsauttaminen</strong> (tai Enter kohdistetulla paikalla) avaa siihen saman kortin kuin pöytäruudukossa: käynnissä olevan ottelun voittaja kirjataan kahdella napsautuksella, ja päättynyt ottelu korjataan napsauttamalla oikean voittajan nimeä.</p>
<p><strong>Ottelut</strong>-välilehti (otsikkona ”Paikat”) yhdistää turnauksen kirjastoon. Jokainen turnauksen ottelu on paikka, joka täytetään kahdella tavalla: litteroimalla ottelu heti (Litterointipaneeli) tai liittämällä siihen jo tuotu ottelu. <strong>Mitään ei liitetä päättelemällä</strong>: nimien yhteensattuma on ehdotus hyväksyttäväksi, osittaista osumaa ei edes ehdoteta, ja jos liitetyn ottelun tiedosto on ristiriidassa kirjatun tuloksen kanssa, ero näytetään ratkaisematta sitä — turnauksen aikana johtajan sana pätee.</p>
<p><strong>Sijoitukset</strong>-välilehti näyttää nykyiset sijoitukset osio kerrallaan, kunkin pelaajan tuloksineen (voitot–tappiot) ja palkintoineen, kun palkintopotti on asetettu. Kaksi tasapisteissä olevaa jakavat sijan ja palkinnon. Poistunut pelaaja säilyttää sijan, jonka hänen tiensä hänelle antaa, merkinnällä ”poistui” ja tuloksellaan tai sillä kaavion kohdalla, johon hän jäi. <strong>Sulje turnaus</strong> lukitsee loppusijoituksen. Sijoitukset kopioidaan CSV:nä käyttöliittymän kielellä, osion ja kunkin viimeisimmän vaiheen, johon hän on päässyt, kera tai tallennetaan tiedostoon: <strong>Tallenna…</strong> avaa järjestelmän valintaikkunan ehdotetulla nimellä, turnaus sekä sana ”sijoitukset” ja päivän päivämäärä, ja tiedosto sisältää täsmälleen kopioidun CSV:n. Komentorivillä <code>blunderdb tournament standings</code> kirjoittaa saman CSV:n. Pelaajan nimi on linkki: se avaa pelaajaan rajatun Historian, jossa jokainen hänen tuloksensa voidaan korjata.</p>
<p>Sulkeminen ilman käynnissä olevia otteluita on yksi napsautus; kun kaikki on pelattu, jono tarjoaa myös <strong>Sulje turnaus</strong> -ehdotuksen, joka käynnistetään kuten muutkin ehdotukset. Jos otteluita on käynnissä, Sijoitukset kertoo niiden määrän ja odottaa toista napsautusta paikan päällä: sulkeminen lukitsee sijoitukset ilman niitä, eikä niiden tulosta voi enää kirjata. <strong>Avaa uudelleen</strong> vahvistetaan samalla tavalla; lopullinen sijoitus lakkaa silloin olemasta lopullinen, ja uudelleenavaus jää lokiin.</p>
<p><strong>Historia</strong>-välilehti on loki selkokielisenä: rivi päätöstä kohti, järjestyksessä, suodatettavissa pelaajan tai ottelun mukaan. Sitä johtaja lukee uudelleen kiistan jälkeen, ja siellä vanhempi päätös korjataan tai kommentoidaan. Tulosrivi nimeää molemmat pelaajat: voittajan ja hänen vastustajansa.</p>
<p>Tuloksen rivillä <strong>Korjaa</strong> avaa sen alle saman korjauksen kuin viimeisimmälle päätökselle: napsauta oikeaa voittajaa, tarvittaessa tuloksen kanssa. Alkuperäinen tulos pysyy paikallaan lokissa, korjaus lisätään siihen, ja sarjataulukko ottaa sen heti huomioon.</p>
<h4>Asetukset</h4>
<p><strong>Asetukset</strong>-välilehti avautuu <strong>nimettyihin muotoihin</strong>: kuusi käyttövalmista seuraturnausta, joista ensimmäistä suositellaan. Yhden valitseminen riittää aloittamiseen; kentät pysyvät muokattavina jälkeenpäin.</p>
<p>Täällä asetetaan: vaiheet (<strong>Lisää vaihe</strong> lisää sen muiden perään; jo avattua vaihetta ei voi poistaa) ja niiden ottelupituus, finaalin pituus, kaavion pituudet kierroksittain (”15, 13, 11” luetaan viimeisestä kierroksesta taaksepäin), <strong>loppupituus</strong> (pidemmät ottelut, kun elossa olevia pelaajia on enää tietty määrä), <strong>mikrokierrokset</strong> (parita erissä N minuutin välein), pöytien määrä, suunniteltu tahti minuutteina pistettä kohden (oletus 8; kellopalkki ja arvioitu loppu lähtevät siitä), päivän tauot, palkintopotti (osallistumismaksu, seuran pidätys, asteikko osioittain) ja näyttökansio.</p>
<p><strong>Kaaviolla</strong> on kolme valintaruutua: <strong>Lohdutuskaavio</strong> (sen hävinneet pelaavat toisen kaavion, oma osionsa sijoituksissa), <strong>Sovittelu</strong> (lohdutuskaavion voittaja pelaa pääkaavion voittajaa vastaan; ruutu näkyy vain lohdutuskaavion kanssa) ja <strong>Lataus</strong> (tuplapudotuksessa pääkaavion voittaja on voitettava kahdesti; ruutu näkyy vain sovittelun kanssa). Lohdutuskaaviolla on omat sijoituksensa vain palkintoasteikon kanssa: niin kauan kuin osion <em>Lohdutuskaavio</em> asteikko on tyhjä, Asetukset muistuttavat siitä. <strong>Lohkot</strong>-vaihe asetetaan lohkojen koolla (oletus 4) ja jatkoon pääsevien määrällä lohkoa kohden (oletus 2).</p>
<p>Sveitsiläisen vaiheen <strong>vaihtoraja</strong> on jäljellä olevien elämien summa, jonka kohdalla siirrytään kaavioon. Jos alkuelämien summa (pelaajat × elämät) on jo pienempi tai yhtä suuri kuin vaihtoraja, sveitsiläinen vaihe ohitettaisiin: Asetukset huomauttavat siitä ennen ensimmäistä käynnistystä.</p>
<p>Asetukset ovat käytettävissä <strong>turnauksen aikana</strong>: vaihtorajan laskeminen kello 22 aiemmin lopettamiseksi tai lohdutusturnauksen lisääminen lauantai-iltana, kunhan kaaviota ei ole arvottu. Se, mikä on silloin lukittu, näkyy harmaana syineen: avatun vaiheen formaatti, sen jakamien elämien määrä ja vaiheen arvonnasta alkaen sen lohkojen koko ja jatkoon pääsevien määrä. Tallennus turnauksen aikana näyttää ensin luettelon muutoksista ja pyytää vahvistusta. Se, minkä moottori hylkää, näkyy siinä syineen, eikä silloin tallenneta mitään: näin käy jo arvotun kaavion lohdutusturnaukselle, sovittelulle tai lataukselle.</p>
<p>Pöytä, jonka lauta on rikki, ilmoitetaan kohdassa <strong>Käytöstä poistetut pöydät</strong>: sen numerot pilkuilla eroteltuina (”7, 12”). Moottori ei enää jaa sitä, ja ruudukko näyttää sen poissa käytöstä; pöytien määrän laskeminen poistaisi viimeisen pöydän, ei rikkinäistä. Jos käytöstä poistetussa pöydässä on ottelu käynnissä, muutosluettelo kertoo sen ja nimeää vapaan pöydän, johon sen voi siirtää ottelukortista.</p>
<p><strong>Sijoitukset</strong> (seeding) on valinta, oletuksena pois: moottorin tutkimus päätyy siihen, ettei suojattuja sijoituksia käytetä, mikä on backgammonin nykyinen tapa. Päällä pelaajat asetetaan luokituksen mukaan.</p>
<h4>Tapahtuman kilpailut</h4>
<p>Useat samoilla pöydillä pelattavat kilpailut — pääkilpailu, speed, nelinpeli — kootaan <strong>tapahtumaan</strong> Asetusten alussa (supistettuna otsikkoonsa, kun kilpailu on liitetty siihen; <strong>Tapahtuman asetukset</strong> -painike <strong>Kaikki pöydät</strong> -välilehdellä johtaa sinne): <strong>Luo ja liitä…</strong> luo tapahtuman pöytämäärineen, <strong>Liitä…</strong> lisää siihen johdetun kilpailun. Liittäminen näyttää ensin, mikä muuttuu: kilpailun pöydät muuttuvat tapahtuman pöydiksi. <strong>Irrota tapahtumasta</strong> palauttaa kilpailun itselleen lokeineen ja pöytineen; <strong>Poista tapahtuma</strong> siirtää sen vahvistuksen jälkeen roskakoriin ja irrottaa sen kilpailut poistamatta yhtäkään.</p>
<p>Tapahtumassa mikään kilpailu ei ehdota pöytää, jossa toinen pelaa: ruudukko näyttää nämä pöydät varattuina kilpailun nimen kanssa, ja paritus ilman vapaata pöytää odottaa. Käytöstä poistettu pöytä merkitään kerran, tapahtumassa, ja se koskee kaikkia sen kilpailuja; yhden kilpailun Asetuksissa muutettu pöytämäärä, käytöstä poistetut pöydät ja tauot koskevat myös tapahtumaa, ja muutosluettelo nimeää muut kilpailut.</p>
<p><strong>Pöytien ominaisuudet</strong> asetetaan Tapahtuma-paneelissa: taulukko, jossa on yksi rivi pöytää kohti, pöydän <strong>nimi</strong> (esimerkiksi ”Stream”), <strong>sali</strong> (vapaa nimike), <strong>Varattu</strong>-valintaruutu ja pelaajat, joille pöytä on <strong>Osoitettu</strong>, valittuina tapahtuman kilpailuihin ilmoittautuneiden joukosta. Neljänkymmenen pöydän kohdalla <em>Pöydät N–M, sali</em> asettaa salin koko välille yhdellä kertaa; <strong>Tallenna</strong> kirjoittaa vain pöydät, joilla on jokin ominaisuus, muut pysyvät tavallisina pöytinä. Varattua pöytää ei koskaan ehdoteta, mutta sille voi silti sijoittaa ottelun käsin käynnistyksellä, siirrolla tai vetämällä ja pudottamalla. Osoitettu pöytä saa ensin haltijansa ottelun, kun se on vapaa; muuten ottelu saa tavallisen pöydän, ja haltijoidensa otteluiden ulkopuolella se käyttäytyy kuin varattu pöytä. Kaksi eri pöytien haltijaa, jotka kohtaavat, pelaavat kahdesta pienemmällä.</p>
<p><strong>Sali</strong> on joukko pöytiä, joilla on sama nimike: ”sali A” pöydille 1–20, ”sali B” seuraaville. Kunkin liitetyn kilpailun Asetuksissa <em>Salit, joissa tämä kilpailu pelataan</em> valitsee salit, joissa se pelataan: DMP salissa B, speed salissa A. Jos mitään ruutua ei ole valittu, kyse on kaikista pöydistä; kilpailu ei saa ehdotuksia saliensa ulkopuolelle eikä voi siirtää sinne ottelua. Sellaisen salin poistaminen, jossa on kilpailun käynnissä oleva ottelu, evätään, ja pöytä nimetään. Yksin pelattava kilpailu asettaa samat pöytien ominaisuudet omissa Asetuksissaan, ilman kilpailun saleja.</p>
<p><strong>Kauden sijoitus</strong> Tapahtuma-paneelissa laskee yhteen tapahtuman päättyneet kilpailut: kukin sija tuottaa <strong>Pisteytyksen</strong> pisteet (voittaja ensin, oletuksena 25, 18, 15, 12, 10, 8, 6, 4, 2, 1), ja tasapisteissä olevat jakavat keskenään haltuunsa ottamiensa sijojen keskiarvon. Henkilö tunnistetaan kilpailusta toiseen nimen perusteella. <strong>Seuran Elo</strong> lisää sarakkeen: jokainen aloittaa 1500:sta ja kauden ottelut pelataan uudelleen järjestyksessä FIBS-kaavan mukaan. <strong>Laske</strong> näyttää sijoituksen, <strong>Kopioi CSV:nä</strong> kopioi sen yhdellä pistesarakkeella kilpailua kohti. Päättymätön kilpailu ei tuota mitään.</p>
<p>Tapahtuman kilpailun Johdon avaaminen avaa myös muut: Johdon yläreunaan ilmestyy välilehti jokaista kilpailua kohti, kussakin oma yhteenveto — odottavat ehdotukset, käynnissä olevat ottelut, hälytys jos sellainen on. Kilpailun vaihtaminen on yksi napsautus sen välilehteen, ilman vahvistusta; jätetty kilpailu ei sulkeudu eikä toista mitään, se pysyy juuri sellaisena kuin se jätettiin. Tapahtuman ulkopuolisella turnauksella on vain yksi kilpailu: ei välilehteä näytettäväksi. Kun vaihdat sovelluksen toiseen välilehteen ja palaat Turnauksiin, Johto palautuu sellaisena kuin sen jätit: sama kilpailu tai <em>Kaikki pöydät</em>, ja sama näkymän välilehti.</p>
<p>Välilehti <strong>Kaikki pöydät</strong>, kilpailujen vasemmalla puolella, näyttää tapahtuman kaikki pöydät yhdessä ruudukossa: yksi ruutu pöytää kohden, mikä kilpailu sitä sitten pitääkin, merkittynä kilpailunsa nimellä ja värillä. Tuloskortti, pikavalikot, näppäimistö ja vedä ja pudota toimivat siellä kuten kilpailun ruudukossa, ja jokainen ele kohdistuu ruudun kilpailuun. Ottelun vetäminen toisen kilpailun pöydälle vaihtaa ottelut keskenään vahvistuksen jälkeen, joka nimeää molemmat kilpailut. Ruudukon alla kaikkien kilpailujen ehdotukset muodostavat yhden jonon, kukin merkittynä kilpailullaan ja omalla <em>Käynnistä</em>-painikkeellaan: pisimpään odottaneet pelaajat tulevat ensin, joten kilpailu ei odota toisen kierroksen loppua. Ottelu ilman pöytää — kaikki pöydät varattu tai pelaaja kiinni toisen kilpailun ottelussa — seuraa jonossa syineen ilman <em>Käynnistä</em>-painiketta, kunnes pöytä tai pelaaja vapautuu. Kilpailun tai näkymävälilehden napsauttaminen poistuu <em>Kaikista pöydistä</em>.</p>
<p>Kun tapahtumalla on useita saleja, ruudukko ryhmitellään saleittain kunkin nimen alle. Pöydän nimi näytetään sen numeron vieressä, varatulle pöydälle lippu ja osoitetulle pöydälle tähti, jota seuraavat haltijat; myös ehdotukset nimeävät pöydän.</p>
<p>Sama henkilö voi pelata useita tapahtuman kilpailuja: kaksi samannimistä osallistujaa ovat sama henkilö, ja nelinpeliparissa kumpikin jäsen lasketaan. Niin kauan kuin hän pelaa yhdessä kilpailussa, muut eivät ehdota häntä, ja niiden <em>Odottaa</em>-lista kertoo, missä hän pelaa: ”pelaa pääkilpailussa, pöytä 4”. Häntä odottava kaavion ottelu jää jonoon, ja sen käynnistäminen hylätään niin kauan kuin hän pelaa. Käsin pariuttaminen on edelleen sallittua: ottelu alkaa, ja sen ruudussa on sama merkintä.</p>
<h4>Turnauksen näyttö</h4>
<p>Turnausta katsotaan. <strong>Näyttökansion</strong> valitseminen asetuksista riittää kertaheitolla: blunderDB kirjoittaa sinne itsenäisen HTML-sivun uudelleen jokaisen muutoksen kohdalla, ja sivu latautuu itsestään. Se aukeaa offline-tilassa, toisella näytöllä tai heijastettuna, eikä lataa ulkopuolisia resursseja. <em>Avaa selaimessa</em> näyttää sen heti.</p>
<p><strong>Parituslomake</strong> asetetaan ilmoittautumispöydälle: yksi napsautus <em>Tulosta lomake</em> avaa järjestelmän tulostusikkunan. Rivi ottelua kohti — kaksi pelaajaa, pituus, pöytä, kaksi tyhjää ruutua tulokselle — ja kolmenkymmenenkahden pelaajan kierros mahtuu yhdelle A4-sivulle.</p>
<p>Tapahtumalla on oma tulostuskansionsa, valittu kerran sen Asetuspaneelissa samalla <em>Valitse kansio</em> -painikkeella: blunderDB kirjoittaa sinne <code>index.html</code>-tiedoston, tapahtuman <strong>seinäsivun</strong> — yksi rivi per pöytä, riippumatta siitä mikä kilpailu sitä käyttää, kunkin kilpailun ilmoitetuilla kierroksilla ja linkillä sen omalle sivulle — ja jokainen liitetty kilpailu kirjoittaa omansa alikansioon. Toiminto missä tahansa tapahtuman kilpailussa uudistaa seinäsivun; liitetyn kilpailun oma kansio säilyy mutta jää huomiotta niin kauan kuin se pysyy tapahtumassa.</p>
<p>Kun kilpailu on pudotuspelivaiheessa ja pudotuspelikaavio on arvottu, sen seinäsivu ja tapahtuman seinäsivu näyttävät puun suurena, kaukaa luettavana: yksi sarake kierrosta kohden, ja lohdutussarjaan putoavat häviäjät katkoviivalla. Sivu vaihtaa näkymää itsestään ilman skriptiä tavallisen sisältönsä (tapahtumalla pöydät) ja jokaisen pudotuspelivaiheessa olevan kilpailun puun välillä, kahdentoista sekunnin ajan kutakin; se latautuu aina uudelleen kolmenkymmenen sekunnin välein ja jatkaa kiertoa siitä, mihin se jäi. Kilpailulla, jolla ei ole pudotuspelikaaviota — esimerkiksi sveitsiläisellä vaiheella — ei ole puuta, ja sivu pysyy entisellään.</p>
<p>Otsikon ”Pelaanko?” alla kilpailun sivu ja tapahtuman seinäsivu nimeävät pelaajat, jotka eivät pelaa juuri nyt, siten kuin moottori ne sijoittaa: ”pudonnut”, kun pelattavia otteluita ei enää ole, ”jatkoon” sen vaiheen nimen kanssa, johon pelaaja siirtyy, ”ei vielä ratkennut”, kun hänen kohtalonsa riippuu vaiheen päättymisestä tai varasijasta, jota johtaja ei ole ratkaissut, ”vapaakierros — tulee mukaan kierroksella N” pelaajalle, joka on arvottu kaavioon ilman vastustajaa, kunnes hän on pelannut, ”vapaakierros — pelaa taas kierroksella N” sveitsiläisen kierroksen vapaalle, ja ”voittaja”. Turnauksen päätyttyä tilanne luetaan sarjataulukosta.</p>
<p>Käyttöliittymän ulkopuolella alikomento <code>blunderdb tournament</code> lukee johdetun turnauksen ilman graafista käyttöliittymää: <code>list</code>, <code>verify</code>, <code>standings</code>, <code>page</code> ja <code>export</code>; <code>proposals</code> näyttää moottorin numeroidun jonon, ja <code>confirm</code> vahvistaa yhden sen kohdista, varasija mukaan lukien; <code>ranking --season</code> laskee yhteen tapahtuman tai ajanjakson suljetut turnaukset kauden sijoitukseksi sijakohtaisella pisteytyksellä ja halutessa seuran Elolla (kaksi samannimistä osallistujaa samassa suljetussa kilpailussa saa sijoituksen hylätyksi, koska se tunnistaa henkilön nimestä); <code>page --rencontre</code> kirjoittaa tapahtuman seinäsivun yhden kilpailun sivun sijaan; <code>hall</code> tulostaa tapahtuman <em>Kaikki pöydät</em> -ruudukon ja sen jonon, <code>tables</code> pöytien ominaisuudet ja kilpailujen salit. Katso Komentoriviliittymä (CLI).</p>
<h3>Stats-paneeli</h3>
<h4>Johdanto</h4>
<p><strong>Stats-paneeli</strong> mahdollistaa oman pelitason analysoinnin ja kehityksen seuraamisen ajan myötä tietokantaan tuotujen asemien perusteella. Se laskee ja näyttää tunnusluvut <strong>PR</strong> (<em>Performance Rating</em>) ja <strong>MWC cost</strong> (Match Winning Chance cost) kaikille asemille tai suodatetulle osajoukolle.</p>
<p>Stats-paneeli on erityisen hyödyllinen seuraaviin tarkoituksiin:</p>
<ul>
<li><strong>oman tason arviointi</strong> tasovyöhykkeisiin nähden (<em>Maailmanluokka</em>, <em>Ekspertti</em>, <em>Edistynyt</em>…) kokonais-PR:n avulla;</li>
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
<li><strong>Moottori</strong> ja <strong>Vähimmäissyvyys</strong> — säilytä vain tietyn moottorin (gnubg, xg…) analysoimat päätökset tai ne, jotka on analysoitu vähintään tietyllä syvyydellä, plyinä.</li>
</ul>
<p><strong>↺ Palauta</strong>-painike tyhjentää kaikki suodattimet (paitsi automaattisesti tunnistetun pelaajan).</p>
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
<p>Virheiden vuoksi menetetty kumulatiivinen ottelun voittotodennäköisyys koko suodatetussa aineistossa. Laskettu tietokannan nykyisellä MET:llä, oletuksena blunderDB:hen sisältyvällä Kazaross-XG2:lla. Toisella taulukolla laskettu ottelun pistetilanteen analyysi on ”eri MET” ja jää tilastojen ulkopuolelle.</p>
<div class="admonition caution">
<p>MWC cost <strong>ei sovellu</strong> <em>money-game</em> -asemiin (joissa ei ole ottelupanosta). Nämä asemat jätetään pois MWC-laskennasta. MWC-arvot riippuvat käytetystä MET:stä; ne eivät ole suoraan vertailukelpoisia eri MET:ejä käyttävien ohjelmistojen välillä.</p>
</div>
</blockquote>
<p>PR ↔ MWC -vaihto on välitön: backend ei suorita uudelleenlaskentaa.</p>
<h4>HTML-raportti</h4>
<p>Paneelin otsikon <strong>HTML-raportti</strong>-painike tuottaa <strong>itsenäisen</strong> asiakirjan: yksi tiedosto, ei ulkoista kuvaa, ei etätyylitiedostoa, ei skriptiä. Kaaviot ovat upotettua SVG:tä, piirretty samalla piirtimellä kuin lauta näytöllä, sinun paletillasi. Se aukeaa missä tahansa selaimessa, kulkee sähköpostitse ja <strong>tulostuu PDF:ksi itse selaimesta</strong> — mikä säästää PDF-generaattorin mukaan ottamiselta sellaisen tuottamiseen, joka kaikilla jo on.</p>
<p>Se sisältää nykyisen alueen tunnusluvut (asemat, ottelut, lasketut päätökset, kokonais-, siirto- ja kuutio-PR), sitten <strong>kymmenen kalleinta päätöstä</strong>, kukin kaavionsa, kustannuksensa, sen ottelun josta se tulee, ja parhaan siirron kun analyysi sen antaa.</p>
<p>Asiakirjan rakentaa moottori, ei näyttö: komentorivi (<code>stats report --html</code>, katso stats — Toistuvat virheet) ja HTTP-palvelu (reitti <code>stats.report</code>) tuottavat saman raportin jollakin käyttöliittymän yhdeksästä kielestä. Vain graafinen sovellus piirtää kaaviot laudan paletilla; kaksi muuta käyttävät oletuspalettia.</p>
<p>Raportti kantaa Tilastot-paneelin <strong>nykyistä suodatinta</strong>. Raportti joka ei kerro aluettaan on raportti jonka luvut eivät merkitse mitään: aseta suodatin — turnaus, päivämääräväli, pelaaja — ennen kuin tuotat sen.</p>
<h4>Yleiskatsaus-välilehti</h4>
<p><strong>Yleiskatsaus</strong>-välilehti antaa yhteenvetonäkymän keskeisistä tunnusluvuista.</p>
<h5>Tasokortit</h5>
<p>Kolme korttia näyttää PR:n (tai MWC:n) seuraaville:</p>
<ul>
<li><strong>PR Yhteensä</strong> — kaikki päätökset (nappulasiirrot + kuutio);</li>
<li><strong>Pelinappula-PR</strong> — vain nappulapäätökset;</li>
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
<h5>Toistuvat virheet</h5>
<p>Välilehden alussa taulukko ryhmittelee nykyisen suodattimen virheet — eli yhden pelaajan virheet, kun pelaaja on suodatettu — <strong>pelisuunnitelman</strong> ja <strong>teeman</strong> mukaan, kallein ensin. Se vastaa kysymykseen ”missä menetän eniten?”: esimerkiksi ”pito · liikaa blotteja”.</p>
<ul>
<li><strong>Virhe</strong> on laskettu päätös, jonka kustannus yltää kirjaston <em>Virhe</em>-rajaan.</li>
<li><strong>Pelisuunnitelma</strong> on vuorossa olevan pelaajan, sellaisena kuin Erittelyt-välilehti sen esittää.</li>
<li>Nappulasiirron <strong>teema</strong> on se, jonka Analyysipaneelin selityslauseen säännöt nimeävät: gammon aliarvioitu, liikaa blotteja, pistettä ei tehty, liian passiivinen. Kuutiopäätöksen teema on virheen suunta, kuten jäljempänä kuutiovirheiden suunnassa: ohitettu tuplaus, ennenaikainen tuplaus, virheellinen pass, virheellinen take.</li>
<li>Virhettä, jota mikään sääntö ei nimeä varmasti, ei arvata: se jää pois luokituksesta ja näkyy erikseen taulukon alla, yhtenä rivinä pelisuunnitelmaa kohti (»tunnistamatonta teemaa ei ole: N virhettä, hinta X»), sekin napsautettavana. Tämä jäännös on usein raskain, koska selitys ottaa kantaa vasta 60 mp:stä alkaen, <em>Virhe</em>-kynnyksen yläpuolella: muiden joukossa luokiteltuna se johtaisi taulukkoa, joka ei kertoisi mitään.</li>
<li><strong>Kustannus</strong> on osuus suodattimen PR:stä, jonka ryhmä edustaa: ryhmän virheisiin sovellettu PR-kaava suhteutettuna kaikkiin laskettuihin päätöksiin. Ryhmien kustannukset eivät siis koskaan ylitä PR:ää.</li>
</ul>
<p>Ryhmää napsauttamalla ladataan sen asemat, kalleimmasta halvimpaan. Teema lasketaan uudelleen jokaisella näytöllä eikä sitä tallenneta koskaan: pelisuunnitelman tavoin se on johdettu tunniste, jota ei voi muokata. Komentoriviltä: <code>blunderdb stats recurring</code> (ks. stats — Toistuvat virheet).</p>
<p>Jokainen rivi tarjoaa kolme tapaa siirtyä virheestä opiskeluun: <strong>Tietovisa tästä ryhmästä</strong> käynnistää Harjoittelu-paneelin Päätös-harjoituksen ryhmän asemilla, <strong>Anki-pakka</strong> tekee niistä korttipakan, <strong>Kokoelma</strong> tallentaa ne uuteen kokoelmaan. Taulukon yläpuolella <strong>Tietovisa kolmesta pahimmasta ryhmästäni</strong> arpoo kaksikymmentä asemaa kolmen kalleimman ryhmän asemista. Komentorivillä <code>stats recurring --quiz</code> arpoo nämä asemat ja <code>--deck</code> luo pakan.</p>
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
<h4>Harjoittelu ajan myötä</h4>
<p><strong>Harjoittelu</strong>-välilehti asettaa rinnakkain samoissa kalenteri-ikkunoissa — valintasi mukaan <strong>viikko</strong> tai <strong>kuukausi</strong> — kolme sarjaa, jotka mittaavat edistymistä kolmella tavalla:</p>
<ul>
<li><strong>visan PR</strong>: Harjoittelu-paneelin Päätös-harjoituksen istuntojen PR, painotettuna arvioitujen päätösten määrällä. Se lasketaan todellisen PR:n asteikolla, joten se on vertailukelpoinen sen kanssa;</li>
<li>nykyisen suodattimen <strong>otteluiden PR</strong>, painotettuna päätösten määrällä;</li>
<li><strong>Ankin pysyvyys</strong>: jo opittujen korttien kertausten osuus, jotka on arvioitu <em>Vaikea</em> tai paremmaksi, luettuna oikealta akselilta (%).</li>
</ul>
<p>Kaavion alla <strong>visan PR pelisuunnitelman mukaan</strong> listaa kirjaston asemista arvotut visan päätökset, huonoimmat suunnitelmat ensin, sekä viimeisen ikkunan PR:n, jossa suunnitelmaa pelattiin.</p>
<p>Sulkeissa kunkin arvon takana oleva päätösten tai kertausten määrä: ikkunalla ilman otosta ei ole arvoa — viiva, ei nolla. Suodatin rajaa vain ottelut; visan ja Ankin lokit ovat omiasi eivätkä sisällä pelaajaa. Mitään ei tallenneta lisää: kolme sarjaa luetaan olemassa olevista lokeista. Komentorivillä: <code>blunderdb stats training</code> (katso stats — Toistuvat virheet).</p>
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
<p>Viisi edellistä välilehteä kuvaavat <strong>yhtä</strong> pelaajaa; <strong>Pelaajat</strong>-välilehti vertaa kaikkia. Se näyttää yhden rivin kutakin tietokannan pelaajaa kohti, mikä vastaa koko kilpailua seuraavan järjestäjän tarpeeseen yksittäisen pelaajan sijaan.</p>
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
<td>Nimi <strong>sellaisena kuin se otteluissa esiintyy</strong>. Kahdella kirjoitusasulla tallentunut pelaaja näkyy kahdella rivillä, ellei toinen ole toisen alias (asetusten Korpus-osio): rivillä on silloin kanoninen nimi.</td>
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
<td>Kirjaston blunderin kynnykseen yltävien virheiden määrä (oletuksena 0,100 EMG).</td>
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
<li><strong>Avaa pelaajan tiedot</strong> — napsauta riviä. Pelaaja valitaan suodatinpalkissa ja näkymä vaihtuu Yleiskatsaus-välilehdelle.</li>
<li><strong>Rajaa ajanjaksoa</strong> — päivämäärä-, turnaus- ja ottelupituussuodattimet toimivat tavalliseen tapaan, joten taulukon voi rajata yhden kilpailun päiviin.</li>
<li><strong>Vertaa kahta pelaajaa</strong> — rastita ensimmäisen sarakkeen ruutu kahdella rivillä. Taulukon yläpuolelle ilmestyy lohko, joka asettaa heidän lukunsa vastakkain; kolmannen pelaajan rastittaminen korvaa vanhemman kahdesta. Ruutu ei valitse riviä: rastitus vertaa, napsautus avaa tiedot.</li>
<li><strong>Katso paikat, joissa toinen pelasi paremmin</strong> — vertailulohkon painike listaa paikat, jotka molemmat pelaajat joutuivat pelaamaan ja joissa toinen pelasi hyvin (kirjaston Virhe-kynnyksen alapuolella) ja toinen ei, suurin ero ensin; pelaaja arvioidaan paikan huonoimman siirtonsa mukaan. <em>Avaa nämä paikat</em> lataa ne analyysinäkymään. Komentorivi ja palvelin antavat saman listan (<code>stats contrast</code>).</li>
<li><strong>Pisteytä siirrot ensin</strong> — luettelo vertaa vain siirtoja, joiden virhe on pisteytetty, eikä mikään tuonti pisteytä niitä: pisteytys tehdään pyynnöstä. Niin kauan kuin pisteytettäviä on jäljellä, lohko näyttää niiden määrän ja painikkeen <em>Pisteytä siirrot</em>; komentorivi tekee saman komennolla <code>repair --move-errors</code>.</li>
</ul>
<p>Lohkossa <strong>vain suhdeluvut saavat tuomion</strong>, ja parempi kahdesta lihavoidaan. Kolme lukua ei saa sitä koskaan, ja syy kannattaa sanoa. <strong>Onni</strong> ei ole ansio: onnekkaampi pelaaja ei ole parempi. <strong>Ottelut, tulos ja päätökset</strong> kertovat mitä suhdeluvut ovat arvoltaan, mutta niiden kilpailuttaminen antaisi voiton sille, joka vain pelasi enemmän. <strong>Blundereiden määrää</strong> ei verrata raakana — kaksitoista tuhannesta päätöksestä on parempi kuin kymmenen sadasta —, joten lohko lisää rivin <em>Blunderit / 100 päät.</em>, joka on vertailukelpoinen, ja jättää määrän viereen taustaksi.</p>
<p>Tasatulos ei ole voitto: sitä ei lihavoida kummallakaan puolella. Suhdeluku, jonka takana ei ole mitään, näkyy merkkinä ”—” eikä ratkaise mitään.</p>
<div class="admonition note">
<p>Tällä välilehdellä <strong>Pelaaja</strong>-luettelo ja <strong>päätöstyypin</strong> valinta ovat poissa käytöstä: taulukko näyttää kaikki pelaajat ja erittelee nappula- ja kuutiopäätökset jo omiin sarakkeisiinsa.</p>
</div>
<p>Paneelin <strong>Corpus</strong>-välilehdellä, komentorivillä ja daemonin API:n kautta (katso stats — Toistuvat virheet) on kolme korpusnäkymää: kahden pelaajan <strong>kasvokkain</strong>-vertailu (yhteiset ottelut, kummankin PR, tase), liukuva <strong>PR kalenteri-ikkunassa</strong> (1, 3, 6 tai 12 kuukautta) ja <strong>sijoitus</strong> PR:n mukaan niille pelaajille, joilla on vähintään tietty määrä laskettuja päätöksiä. Corpus-välilehti laskee kunkin näkymän pyydettäessä nykyisen suodattimen alla. <strong>Alkuperäsuodatin</strong> (suodatinpalkin kentät <em>Moottori</em> ja <em>Vähimmäissyvyys</em>) rajaa tilastot niin analysoituihin päätöksiin; se koskee jokaista päätöstä, ja sen koskettamat luvut lasketaan silloin uudelleen päätöksistä. Kasvokkain-vertailu ja ikkunakohtainen PR eivät hyväksy sitä: niin kauan kuin se on aktiivinen, ne kieltäytyvät laskemasta. Kuten muu palkki, se säilyy istunnosta toiseen.</p>
<div class="admonition important">
<p>Viiva (”—”) merkitsee arvoa, jota <strong>ei ole koskaan mitattu</strong>; sitä ei pidä sekoittaa nollaan. Näin on erityisesti Tuuri-sarakkeen kohdalla kaikissa otteluissa, jotka tuotiin ennen skeemaversiota 2.15.0: tuuria ei silloin tallennettu, eikä mikään salli sen palauttamista jälkikäteen. Lähdetiedoston tuominen uudelleen ei riitä: tuonti tunnistaa kaksoiskappaleen ja ottaa käyttöön vain uudet opiskelumerkinnät (raportti kertoo niiden määrän). Ottelu on poistettava ja tuotava sitten uudelleen. Muodot, jotka eivät sitä kuljeta (BGF, Jellyfish <code>.mat</code>), eivät sitä koskaan tarjoa.</p>
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
<li>Oletuksena MWC cost lasketaan <strong>Kazaross-XG2-MET</strong>:stä, kilpailullisen backgammonin de facto -viitetaulukosta. Tulokset eivät ole suoraan verrattavissa ohjelmistoihin, jotka käyttävät muita METejä. Se on sama taulukko, luettuna saman sisääntulokohdan kautta, jota sisäänrakennettu arviointimoottori käyttää pistetilanteen mukaisiin kuutiopäätöksiinsä: tilastot ja moottori eivät voi poiketa toisistaan tässä. Se antaa omat arvonsa 25 tehtävään pisteeseen asti kummallakin puolella; sen ylitse sitä jatketaan Zadehin taulukolla, joka lasketaan kuten GNUbg:ssä, 64 pisteeseen asti.</li>
<li><em>Money-game</em> -asemat (joissa ei ole ottelun pistetilannetta) <strong>jätetään pois</strong> MWC-laskennasta. Jos tietokantasi sisältää paljon money-game-asemia, MWC cost voi olla aliarvioitu tai ei käytettävissä.</li>
<li>MWC cost on kumulatiivinen koko suodatetussa aineistossa — ei päätöskohtainen tunnusluku. Se mittaa virheidesi kokonaisvaikutusta voittomahdollisuuksiisi.</li>
</ul>
<h3>Eval-paneeli</h3>
<p><strong>Eval</strong>-paneeli (<em>CTRL-E</em>) arvioi reaaliajassa laudalle asetetun aseman, oli se mikä tahansa; bearoff-asemassa se erikoistuu ja laskee lisäksi EPC:n (Effective Pip Count). Se avataan painamalla <em>CTRL-E</em>, napsauttamalla alapaneelin Eval-välilehteä tai suorittamalla komento <code>eval</code>; myös sen vanha nimi <code>epc</code> avaa paneelin. Paneelin nimi oli ensin <em>EPC</em>, sitten <em>Bearoff</em>, ennen kuin siitä tuli <em>Eval</em> — täältä on siis etsittävä sitä, mitä aiempi versio kutsui Bearoff-paneeliksi, sillä tuo nimi tarkoittaa enää bearoff-taulukoiden asetusvälilehteä.</p>
<p>Paneeli näyttää aina sen <strong>ainoan päätöksen</strong>, jota laudalle asetettu asema vaatii — ei koskaan kahta yhtä aikaa — sekä siihen liittyvät faktat. Kukin suure luetaan sille sopivalla akselilla yhden pakotetun akselin sijaan: kummankin pelaajan voitto-, gammon- ja backgammon-todennäköisyys sekä cubeless-ekviteetti, jotka lasketaan <em>ennen heittoa</em>, luetaan <strong>pelaajakohtaisesti</strong> (ala, ylä, sitten Δ) kuutiopäätöksen vasemmalla puolella, kun noppia ei ole asetettu. Faktat ja päätös pysyvät vierekkäin: kuutiopäätös ei koskaan siirry sitä perustelevien lukujen alapuolelle, olipa käyttöliittymän kieli tai laudan asema mikä tahansa. Heti kun noppia asetetaan, nämä samat <em>ennen heittoa</em> -arvot vaihtavat akselia: ne luetaan <strong>vuorossa olevan pelaajan kannalta</strong> ehdokassiirtojen luettelon kärjessä kursivoituna <em>ennen heittoa</em> -rivinä — ei yhtenä ehdokassiirtona lisää, vaan kiintopisteenä, jota vasten kukin siirto luetaan. Tämän rivin ja siirron välinen ero sisältää heiton tuurin, ei koskaan siirron ansiota, eikä sillä siksi ole virhesaraketta. Puhtaassa bearoff-asemassa toinen taulukko, aina <strong>pelaajakohtainen</strong> ja aina läsnä, nopat asetettuina tai ei, sisältää EPC:n, pip countin, wastagen, keskimääräisen heittomäärän ja keskihajonnan; nämä viisi saraketta eivät koskaan siirry. Kaksi taulukkoa on pinottu ja ne jakavat saman sarakeruudukon: samat reunat, samat sarakemerkit, yksi ainoa pistesarake — ne luetaan yhtenä kaksikerroksisena kokonaisuutena. <em>Lisää tietokantaan</em> -painike, tilamerkki, moottorin attribuutio (myös viimeisimmän evaluoinnin syvyys näkyy siinä) ja <em>Haaste</em>-valintaruutu muodostavat erillisen kaistan, joka on tasattu oikealle taulukoiden yläpuolelle.</p>
<p>Vain ehdokassiirtojen luettelo vierii — myös <em>ennen heittoa</em> -rivi pysyy kiinnitettynä sen yläpuolella; muu paneeli (faktat, merkki, kuutiopäätös) pysyy aina näkyvissä ilman erityistä paneelin koon säätöä.</p>
<p>Faktataulukon ja päätöksen laskee sisäänrakennettu gammonNet ilman XG:tä tai gnubg:tä. Laskenta seuraa asemaa jäädyttämättä koskaan käyttöliittymää: 0-ply-syvyys näytetään heti jokaisen eleen jälkeen, ja puolen sekunnin paikallaanolon jälkeen syvempi evaluointi (oletuksena 2 plyä, säädettävissä asetusten <em>gammonNet</em>-välilehdellä) korvaa sen taustalla — mikä tahansa uusi ele peruuttaa tämän taustalaskennan. Merkkikaistalla tai kilpajuoksuasemassa tilamerkin sisällä näytetty syvyys on aina se, joka todella tuotti näytetyn luvun, ei koskaan pyydetty syvyys; sitä ei toisteta joka rivillä, koska reaaliaikainen evaluointi käyttää samaa syvyyttä kaikille siirroille. Ehdokassiirtojen ja kuutiopäätöksen ekviteetti noudattaa aseman pistetilannetta: money gamessa se ilmaistaan pisteinä, ottelun pistetilanteessa <strong>normalisoituna ekviteettinä</strong> — samalla asteikolla kuin XG:ssä ja GNU Backgammonissa, jossa nykyisen kuution arvon voittaminen on +1 ja sen häviäminen −1 — eikä niitä koskaan sekoiteta samassa taulukossa. Sarakkeen otsikko ilmoittaa sen selvästi sen sijaan, että asteikko jätettäisiin arvattavaksi: ”Equity (money)” money gamessa, ”Equity (match)” ottelun pistetilanteessa. Se ottaa huomioon <strong>elävän kuution</strong>: haku arvostaa jokaista loppuasemaa kuutiomallilla (Janowski, mitattu tehokkuus) aseman kuutiotilassa, samaan tapaan kuin XG ja GNU Backgammon tekevät <em>cubeful</em>-arvioinnissa. Tämä tekee gammon-go- ja gammon-save-vaikutukset näkyviksi pistetilanteessa — pistetilanteessa 4-away/2-away jäljessä oleva pelaaja pelaa 8/2 6/2 avaussiirrolla 6-4, koska varhainen tuplaus antaa gammonille ottelun arvon, mitä kuutioton arviointi ei näe. <em>Ennen heittoa</em> -rivi puolestaan pysyy <strong>cubeless</strong>-ekviteettinä: se on aseman fakta, ei päätös. Evaluointia ei koskaan tallenneta: kyseessä on laskenta, ei analyysi. Ehdokassiirron napsauttaminen näyttää sen laudalla nuolina, täsmälleen kuten Analyysipaneelissa. Huomaamaton <strong>?</strong>-painike merkkikaistalla vie moottorin <code>gammonNet &lt;https://github.com/kevung/gammonNet&gt;</code>_ -tietovarastoon; täydellinen attribuutio (Strehlin verkko, gammonNet-kokoonpano) on ohjeen Kiitokset-osiossa.</p>
<p>Merkkikaistan alussa oleva <strong>Lisää tietokantaan</strong> -painike tallentaa laudalla olevan aseman tietokantaan; <em>CTRL-S</em>, komento <code>w</code> ja työkalurivin <em>Tallenna asema</em> -painike tekevät saman. Tallennetaan vain asema, ei koskaan näytettyä evaluointia; jos gammonNetin automaattinen analyysi on käytössä, sen erä käynnistyy sen jälkeen, kuten tuonnin jälkeen. Tilarivi ilmoittaa aseman numeron myös silloin, kun asema oli jo tietokannassa: se merkitään silloin erikseen tuoduksi, ja suodatin <em>Erikseen tuotu</em> (<code>s i</code>) löytää sen. Paneeli pysyy auki samalla laudalla, joka pysyy luonnoslautana: nappulaa voi siirtää ja lisätä muunnelman, ja paneelista poistuminen palauttaa siihen, mitä tutkittiin. Painike on poissa käytöstä niin kauan kuin yhtään tietokantaa ei ole avattu tai asemaa ei voi tallentaa (esimerkiksi paneelin lähtöasema, jossa ylempi pelaaja on pelannut kaikki nappulansa ulos); sen työkaluvihje kertoo syyn.</p>
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
<p><strong>Beaver</strong>-säännön alla rivi <em>tuplaus, otto</em> on vastustajan paras vastaus luovuttamista lukuun ottamatta: otto tai <em>beaver</em> — välitön takaisintuplaus kuutio itsellä pitäen, jolloin peli pelataan nelinkertaisesta panoksesta —, ja tuplaaja vastaa tällöin <em>raccoonilla</em> (kahdeksankertainen panos, kuutio hänellä), kun se kannattaa hänelle. Ratkaisu ja virheet luetaan tältä riviltä kuten ilman sääntöä; ottelun pistetilanteessa sääntö ei ole voimassa.</p>
<p>Kolmas merkki, <strong>Kuution katto</strong>, ilmestyy, kun lähdetunniste rajaa kuution — sekä ottelutilanteessa että money gamessa. Se ei kuvaa yllä näkyvää laskentaa: sisäänrakennettu arviointi ei mallinna kattoa, joten tuomio koskee vapaata kuutiota. Juuri siksi merkki on siinä: katolla rajattu kuutio on ainoa näkyvä syy, jonka takia blunderDB ja eXtreme Gammon voivat ilmoittaa samasta asemasta kaksi eri tuomiota.</p>
<p><strong>Vuorossa olevaa pelaajaa</strong> ja <strong>kuution sijaintia</strong> muokataan suoraan laudalla kuten muokkaustilassa: pelaajan bearoff/pistetilanne-suorakulmion napsauttaminen antaa vuoron hänelle; kuution napsauttaminen kierrättää tilat keskellä → alapelaajan hallussa → yläpelaajan hallussa (oikea painike vastakkaiseen suuntaan). Kuution arvo pysyy kiinnitettynä — money gamessa ekviteetit ilmaistaan nykyisen kuution yksiköissä, vain sen omistajalla on merkitystä. Analyysi lasketaan heti uudelleen. Arvioidussa tilassa itse merkki on napsautettavissa ja avaa suoraan asetusten <em>Bearoff</em>-välilehden; sen työkaluvihje selittää miksi (kuutiopäätöstä ei voi arvioida, <code>ADR-0009 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0009-race-win-chances-are-read-or-convolved-cube-verdicts-are-never-estimated.md&gt;</code>__) ja miten tarkkaa aluetta laajennetaan.</p>
<p>Myös <strong>pistetilannetta</strong> muokataan suoraan laudalla kuten muokkaustilassa: vasen napsautus pelaajan pistetilannesuorakulmioon vähentää hänen tarvitsemiensa pisteiden määrää, oikea napsautus lisää sitä. Kun <em>money</em>-pistetilanteesta (-1, -1) poistutaan muokkaamalla vain toista puolta, toinen puoli tasataan automaattisesti samaan arvoon epäjohdonmukaisen pistetilanteen sijaan. Bearoff-asemassa <em>tarkassa</em> tilassa siirtyminen money-pistetilanteesta ottelun pistetilanteeseen jättää voittotodennäköisyyden ennalleen (tietokannasta luettu arvo, joka pätee viitekehyksestä riippumatta), mutta vaihtaa näytetyn ekviteetin ja kuutiopäätöksen <em>evaluoidun</em> tilan arvoihin — tarkka taulukko on rakenteeltaan money-taulukko, eikä se osaa vastata pistetilanteessa esitettyyn kysymykseen. Merkistä tulee tällöin yhdistelmä (« tarkka (voitto) · evaluoitu (kuutio) »), jotta tämä sanotaan suoraan.</p>
<p>Myös <strong>nopat</strong> muokataan samalla tavalla, ja juuri ne ratkaisevat, mikä kysymys esitetään: asetetut nopat tekevät nappulapäätöksen (ehdokassiirtojen listan), noppien puuttuminen kuutiopäätöksen. Vasen napsautus nopalla kasvattaa sen arvoa (6 palaa 1:een), oikea napsautus pienentää sitä (1 palaa 6:een); nopan napsauttaminen laudalla, jolla ei ole noppia, asettaa kaksi kerralla — yksi noppa ei olisi nappulapäätös eikä kuutiopäätös. Pelaajan suorakulmion napsauttaminen poistaa nopat kuutiokysymyksen esittämiseksi, ja seuraava napsautus nopalla palauttaa ne ennalleen.</p>
<p><em>ASKELPALAUTIN</em> tai kaksoisnapsautus laudan ulkopuolella tyhjentää aseman: tyhjä lauta, money-pistetilanne (-1, -1), ei asetettuja noppia — Eval-paneelin omat arvot, jotka eroavat muokkaustilan arvoista (7 molemmilla, nopat 3-1), jotta ne pysyvät johdonmukaisina paneelin oletusnäytön kanssa.</p>
<h4>Tuplauskuution matriisi</h4>
<p>Kuutiopäätös ei ole laudan ominaisuus. Samat nappulat ja sama pip-luku tuplataan tilanteessa 2-away/4-away eikä tuplata tilanteessa 4-away/2-away; se joka on oppinut money-vastauksen on oppinut yhden ruudun ruudukosta. Eval-paneeli näyttää sen ruudun, jonka asema kantaa; <strong>tuplauskuution matriisi</strong> näyttää koko ruudukon.</p>
<p>Komento <code>cm</code> avaa sen näytöllä olevalle asemalle. Kukin ruutu antaa tuomion yhdessä pistetilanteessa: rivi on vuorossa olevan pelaajan vielä tarvitsemien pisteiden määrä, sarake vastustajan. Neljä tuomiota kirjoitetaan <em>ET</em> (ei tuplausta), <em>TO</em> (tuplaus, otto), <em>TP</em> (tuplaus, pass) ja <em>LH</em> (liian hyvä); moottorin hylkäämässä ruudussa on kysymysmerkki, ja osoitin kertoo syyn sekä ruudun kolme ekviteettiä. Tarjolla on kolme ottelupituutta: 5, 7 ja 9 pistettä.</p>
<p>Ruutu, joka vastaa aseman todellista pistetilannetta, on kehystetty ja sen rivi- ja sarakeotsikot alleviivattu: lukeminen alkaa siitä, ”minun ruutuni ja sen ympäristö”. Kehys ilmestyy heti, kun aseman molemmat <em>away</em>-tilanteet mahtuvat näytettyyn ruudukkoon; pituuden vaihtaminen siirtää kehyksen tai poistaa sen. Money-asema, Crawford-peli tai ruudukon ulkopuolelle jäävä <em>away</em> eivät osoita mitään ruutua: näytettävää ruutua ei ole, ja likimääräisen näyttäminen olisi väärin.</p>
<p>Aseman pistetilanne korvataan kunkin ruudun tilanteella; sen <strong>kuutio</strong> säilyy. Ruudukko vastaa kysymykseen, missä pistetilanteessa kääntäisin <em>tämän</em> kuution, ei siihen mitä keskitetty asema tekisi. Se on kauttaaltaan Crawfordin jälkeinen: Crawford-pelissä kuutio ei ole pelissä, eikä sarake ”et saa tuplata” kertoisi asemasta mitään.</p>
<p>Jokainen ruutu on oma hakunsa. Moottori ottaa pistetilanteen huomioon — se ei pelaa samaa peliä tilanteessa 2-away kuin 7-away — joten yksi ainoa haku luettuna eri otteluekviteettien läpi olisi väärässä juuri siellä, missä pistetilanne merkitsee. Ruudukko saapuu ensin 0-plyllä ja laskeutuu uudelleen määritetyllä näyttösyvyydellä, kun ikkuna on levossa: sama porrastus kuin muualla paneelissa, ja 9 pisteen ruudukko maksaa noin puolitoista sekuntia.</p>
<p>Sama ruudukko lasketaan käyttöliittymän ulkopuolella komentorivin komennolla cubematrix.</p>
<h4>Aseman tuominen Eval-paneeliin</h4>
<p>Paneeli avautuu oletuksena bearoff-asemaan, mutta tutkiminen alkaa useimmiten jo käsillä olevasta asemasta. Kolme elettä tuo sen paneeliin:</p>
<ul>
<li><strong>Oikea napsautus laudalla</strong> analyysipaneelissa tai ottelua selattaessa ja sitten <em>Arvioi tämä asema</em>: Eval-paneeli avautuu suoraan tähän asemaan sellaisena kuin se näytetään; <em>Arvioi tämän aseman peilikuva</em> avaa sen siihen toisen puolen näkökulmasta. Pikavalikko ei ilmesty Eval-paneelissa eikä Hakupaneelissa, joissa oikea painike on jo varattu toisen värin nappuloiden asettamiseen.</li>
<li><strong>CTRL-C ja sitten CTRL-V</strong>: kopioi asema analyysipaneelista ja liitä se sitten Eval-paneelissa. Liittäminen hyväksyy myös muualta tulevan tunnisteen — XGID:n (eXtreme Gammon, GNU Backgammon, toinen blunderDB-instanssi) tai OGID:n (OpenGammon): riittää, että se on leikepöydällä.</li>
<li><strong>Komento</strong> <code>import XGID=…</code> (tai <code>import OGID=…</code>) siihen tapaukseen, ettei tunniste ole leikepöydällä vaan viestissä, päätteessä luetulla foorumilla tai skriptin tuottamana. Se on sama verbi kuin pelkkä <code>import</code>: ilman argumenttia se avaa tiedostovalitsimen, argumentin kanssa se lukee tunnisteen. Polku on sen jälkeen sama kuin liittämisessä — sama luku, sama kaksoiskappaleiden poisto, sama tuodun aseman avaus.</li>
</ul>
<p>OGID kantaa vain aseman: ei arviota eikä kommenttia. Asema saapuu siis ilman analyysiä, aivan kuten paljas XGID, ja sisäänrakennettu arviointi voi täyttää aukon jälkikäteen.</p>
<p>OGID:ssä Crawford-peli tunnistetaan ottelun pituuden perässä olevasta <code>C</code>-kirjaimesta (<code>7C</code>): ilman sitä pelaaja, jolta puuttuu yksi piste voittoon, on jo Crawford-pelin jälkeen.</p>
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
<p><strong>Voittotodennäköisyys ja päätös, arvioitu tila.</strong> Tarkan alueen ulkopuolella voittotodennäköisyys tulee gammonNetin raakatulosteesta (0-ply-haku, sitten asetetulla syvyydellä, ei koskaan taulukosta luettuna) ja päätös tähän tulosteeseen sovelletusta Janowskin « Decide »-mallista — haku <em>pelaa</em> trajektorin sen sijaan, että tiivistäisi siitä hetkellisen tilannekuvan, mikä on täsmälleen se, mitä estimoitu tila ei voinut tehdä (katso alempana), ja mahdollistaa, ainoana kolmesta tilasta tarkan ohella, päätöksen <strong>ottelun pistetilanteessa</strong>.</p>
<p>Tämä tila on mitattu, ei vain oletettu, sisäänrakennettua two-sided-taulukkoa vasten (<code>TestEvalMeasure</code>, 4000 satunnaisotannalla poimittua money-päätöstä, kanoniset parametrit 2 plyä k=12): money-päätösten yhtäpitävyys <strong>93,4 %</strong> (3735/4000), eriteltynä etäisyyden mukaan gammonNetin ottopisteestä — 61,1 % alle 1 %:n päässä ottopisteestä (kolikonheitolle herkin alue), 88,3 % välillä 1–5 %, 91,5 % välillä 5–10 %, 94,0 % välillä 10–20 %, 94,4 % sen yli. Voittotodennäköisyyden ero: keskiarvo 0,85 %, mediaani 0,44 %, 95. persentiili 3,21 %, maksimi 8,30 %. Cubeful-ekviteetin ero: keskiarvo 0,039, mediaani 0,018, 95. persentiili 0,151, maksimi 0,406. Muoto on odotettu: valtaosa erimielisyydestä keskittyy täsmälleen ottopisteeseen, jossa kaksi perustellusti erilaista menetelmää eroavat eniten tiukassa päätöksessä — ei hajanainen virhe, joka maksaisi ekviteettiä kaikkialla.</p>
<p>Tämä mittaus koskee <strong>money</strong>-päätöksiä kilpajuoksussa. Ottelun pistetilanteen mukaisesta tuomiosta — jonka vain tämä tila osaa antaa — ja kontaktiasemista ei ole julkaistua mittausta: edellä sanottu ei siirry näihin tapauksiin.</p>
<p><strong>Miksi ei syvemmälle kuin 2 plytä?</strong> Koska mittaus sanoo, ettei se tuota mitään. Nappulapäätös maksaa samalla koneella 99 ms 2 plyllä ja 8,4 s 3 plyllä — <strong>kahdeksankymmentäviisi kertaa enemmän</strong>. Neljästäkymmenestä molemmilla syvyyksillä toistetusta oikeasta päätöksestä syvempi haku muutti mielensä <strong>kahdesti</strong>, ja molemmilla kerroilla hyöty, jonka se itselleen luki, oli korkeintaan 0,0005 normalisoitua ekviteettiä: kaksi kertaluokkaa alle 0,020:n, sen kynnyksen, jonka jälkeen eXtreme Gammon ylipäätään puhuu virheestä. Päätöstä kohti, kaikki tapaukset yhdessä, hyöty on 0,0000.</p>
<p>2 plyä pysyy siis oletuksena; 3 ja 4 plyä valitaan asetusten <em>gammonNet</em>-välilehdeltä. Tämä ei sano, että 3 plytä olisi yleisesti arvoton, vaan että <em>tällä</em> verkolla, kanonisella suodattimella, se ei maksa paneelin ääressä istuvan odotusta. Mittaus on toistettavissa (<code>TestThreePlyMeasure</code>), ja johtopäätös ratkaistaan uudelleen, jos verkko muuttuu.</p>
<p><strong>Miksi arvioitua päätöstä ei ole?</strong> Seuraava koskee nimenomaan <em>konvoluutio</em>-menetelmää (arvioitu tila), ei yllä kuvattua evaluoitua tilaa: cubeful-ekviteetti on <em>trajektoriongelma</em> (milloin tuplata), jota mikään aseman tilastollinen tiivistelmä ei tavoita — paras mitattu staattinen malli jättää jäännösvirheen (ekviteetin keskihajonta 0,016, maksimi 0,20), joka riittää kääntämään kaikki tiukat päätökset. Samoin päätöksen muuntaminen ottelun pistetilanteeseen match-ekviteettitaulukon avulla mitattiin riittämättömäksi (12 % erimielisyyksiä GNUbg:n 2-ply-analyysin kanssa, mukana todellisia blundereita). Koska itsevarmasti näytetty väärä päätös on pahempi kuin ei päätöstä lainkaan, konvoluutiolla ei ole koskaan ollut oikeutta näyttää päätöstä — tämän aukon täyttää haku, joka pelaa trajektorin, ei tilastollinen tiivistelmä.</p>
<div class="admonition note">
<p>Bearoff-tietokannat ovat muuttumattomia matemaattisia taulukoita. blunderDB laskee ne itse, samoin kuin GNUbg:n <code>makebearoff</code>-työkalu — tavu tavulta — asetusten <em>Bearoff</em>-välilehdellä tai komennolla <code>blunderdb bearoff generate</code>.</p>
</div>
<h3>Anki-paneeli</h3>
<p><strong>Anki-paneeli</strong> (<em>CTRL-K</em>) mahdollistaa asemien opiskelun välitoistolla FSRS-algoritmia käyttäen. Käyttäjä voi luoda pakkoja kokoelmista tai hakutuloksista.</p>
<p><strong>Pakkojen luominen:</strong> Napsauta <strong>+ Uusi pakka</strong> luodaksesi pakan kokoelmasta tai nykyisistä hakutuloksista. Hakuun perustuvat pakat synkronoituvat automaattisesti, kun Anki-välilehti avataan.</p>
<p><strong>Tilannekorttien pakka.</strong> Kolmas lähde, <em>Tilannekortit</em>, ei pyydä muuta kuin nimen: blunderDB täyttää pakan 36:lla järjestämättömällä tilanteella väliltä 2–9 away, ja tilanteen kortti on sama taulukko, jonka Tilanteet-harjoitus näyttää — hyväksymispisteet ja gammon-arvot, molemmat puolet. Tämä pakka on olemassa vain, jos luot sen: 36 ensimmäisenä päivänä erääntyvää korttia on kertausvelkaa, ja se otetaan tietoisesti. Synkronointipainike luo pakan uudelleen.</p>
<p>Nämä kaksi paikkaa eivät tee samaa työtä. Tilanteet-harjoitus panee palauttamaan nämä luvut <strong>kellon käydessä</strong> ja mittaa nopeuden; pakka panee ne <strong>kestämään ajassa</strong> eikä mittaa siitä mitään. Kaksi tarinaa pysyvät erillään: Harjoittelun loki ei näe Anki-kertauksia, eivätkä Ankin tilastot näe harjoitteluistuntoja.</p>
<p><strong>Kertaaminen:</strong> Valitse pakka ja napsauta <em>Opiskele</em> (tai kaksoisnapsauta pakkaa) aloittaaksesi erääntyneiden korttien kertaamisen. Asemakortti näyttää aseman laudalla; tuloskortti ilmoittaa tuloksen ja jättää laudan ennalleen. Arvioi muistamisesi näppäimillä <em>1</em> (Uudelleen), <em>2</em> (Vaikea), <em>3</em> (Hyvä) tai <em>4</em> (Helppo). Paina <em>Esc</em> lopettaaksesi ja palataksesi pakkaluetteloon.</p>
<p>Kahdella luvulla on eri nimet: luettelon sarake <strong>Erääntyneet</strong> laskee kaikki kortit, joiden eräpäivä on mennyt, myös keskeytetyt ja haudatut; <em>Opiskele</em>-painikkeen luku laskee vain tällä hetkellä käytettävissä olevat kortit ja voi siksi olla pienempi.</p>
<p><strong>Kuutiopäätöksistä tulee kaksi korttia, ketjutettuina.</strong> Kuutiopäätös on kaksi kysymystä — ”tuplaus?”, sitten ”hyväksy?” — ja blunderDB on aina tallentanut ne kahtena asemana. Pakka, joka valitsee vain toisen puolikkaan, saa toisenkin: päätös täydennetään, ei laajenneta. Ja kun molemmat ovat vuorossa, toinen tulee <strong>heti</strong> ensimmäisen jälkeen.</p>
<p>Kumpikin säilyttää oman arvosanansa ja oman aikataulunsa: nämä eivät ole yhden kortin kaksi vaihetta, vaan kaksi korttia. Ketjutus ei aikaista mitään eräpäivää — se järjestää jo erääntyneet kortit, ei muuta. Koska ne syntyvät yhdessä, ne erääntyvät yhdessä ensimmäisellä kerralla, ja juuri siinä siitä on hyötyä.</p>
<p><strong>Vastauksen näyttäminen:</strong> Kortti esittää kysymyksen — mikä siirto pelataan tai mikä kuutiotoimi tehdään. Mieti, ja paina sitten <em>VÄLILYÖNTI</em> (tai napsauta peitettyä aluetta) paljastaaksesi vastauksen: aseman tallennetun analyysin sellaisena kuin Analyysi-välilehti sen esittää. Se ilmestyy arviointipainikkeiden alle, jotka pysyvät paikoillaan ja ulottuvilla. Listan siirron napsauttaminen näyttää sen laudalla.</p>
<p>Mikään ei pakota paljastamaan vastausta arviointia varten: jos olet varma, näppäimet <em>1</em>–<em>4</em> pysyvät käytössä. Vastaus peittyy uudelleen seuraavan kortin kohdalla, mutta ei silloin, kun vain vaihdat välilehteä — käy katsomassa Eval-paneelia tai aseman kommenttia, vastaus odottaa palatessasi.</p>
<p>Asema, jolla ei ole tallennettua analyysiä, ilmoittaa sen suoraan ilman peitettyä aluetta.</p>
<p><strong>Vastaaminen laudalla.</strong> Oletuksena arvioit itse itsesi. Valitse asemapakan asetuksissa <em>Vastaa laudalla</em>: nappulakortilla pelaat siirron laudalla kuten Päätös-harjoituksessa ja painat sitten <em>Vahvista</em>. Moottori arvioi siirron tallennettua analyysia vasten, paljastaa vastauksen ja <strong>ehdottaa arvosanaa</strong>: <em>Helppo</em> nopeasta oikeasta vastauksesta, <em>Hyvä</em> hitaammasta oikeasta, <em>Vaikea</em> blunder-rajan alittavasta virheestä, <em>Uudelleen</em> blunderista tai laittomasta siirrosta. Ehdotettu arvosana korostetaan; hallinta pysyy sinulla ja arvioit haluamasi näppäimillä <em>1</em>–<em>4</em>. Laillinen siirto, jota analyysi ei luokittele, ei ehdota mitään. Kuutiokortit, tuloskortit ja tulostaulukkopakat pysyvät itsearvioituina. Vastauksen paljastaminen pelaamatta luovuttaa siirron.</p>
<p><strong>Istunnon rajaaminen.</strong> Oletuksena kertausistunto käy läpi kaikki erääntyneet kortit. Voit rajata sen korttimäärään pakkakohtaisesti Asetuksissa: rastita <em>Rajaa istunto</em> ja ilmoita, montako korttia istunnon tulee tarjota. Kun raja täyttyy, istunto päättyy ja kertoo siitä — viesti erottaa tilanteen ”raja täynnä, näin monta korttia yhä erääntyneenä” aidosti tyhjästä jonosta. Jos haluat silti jatkaa, vapaa harjoittelu on olemassa: se tarjoaa muita asemia muuttamatta aikataulusta mitään.</p>
<p>Raja <strong>0</strong> ei tarjoa yhtään korttia: se on oma tilansa, hyödyllinen pakan jäädyttämiseen turnaukseen valmistautumisen ajaksi, eikä se ole sama asia kuin ”ei rajaa”. <em>Opiskele</em>-painike on tällöin pois käytöstä.</p>
<p>Raja koskee <strong>istuntoa</strong>, ei päivää. blunderDB-pakka rakentuu kokoelman tai haun varaan: se on äärellinen aineisto, joka esitellään muutamassa istunnossa ja jonka päivittäistä määrää sen koko jo rajaa. Päiväkatto ei puraisi koskaan, tai sitten se loisi ruuhkan pakalle, joka mahtui yhteen istuntoon.</p>
<p><strong>Vapaa harjoittelu (cram):</strong> <em>Harjoittele</em>-painike <em>Opiskele</em>-painikkeen vieressä aloittaa vapaan harjoittelusession: sinulle näytetään satunnaisia asemia pakasta FSRS-aikataulusta riippumatta. Tämä tila <strong>ei koskaan muuta kertausaikataulua</strong> — ihanteellinen lämmittelyyn ennen turnausta tai teemapakan tehokkaaseen kertaamiseen ilman, että sen järjestys häiriintyy. <em>Vapaa</em>-merkki korvaa kortin tilan, ja <em>Seuraava</em>-painike (näppäimet <em>1</em>–<em>4</em>) selaa asemia. <em>Esc</em> palaa luetteloon tallentamatta keskeytettyä sessiota.</p>
<p><strong>Kortin siirtäminen syrjään ilman arvosanaa.</strong> Kertauksen aikana kortin otsikon oikea napsautus tarjoaa kolme elettä, jotka poistavat sen istunnosta kertomatta ajoittajalle mitään:</p>
<ul>
<li><strong>Keskeytä</strong> — kortti säilyttää aikataulunsa eikä tule enää esiin niin kauan kuin se on keskeytettynä. Näin siirretään syrjään väärä tai vielä hyödytön kortti menettämättä siihen liittyvää historiaa.</li>
<li><strong>Hautaa</strong> — kortti katoaa seuraavaan päivään asti. Toisin kuin keskeytys, tämä ei sano mitään sen arvosta: se on sille, jonka on juuri nähnyt muualla tai jota ei halua kohdata kahdesti samana iltana.</li>
<li><strong>Poista</strong> — kortti lähtee pakasta vahvistuksen jälkeen. Asema itse jää tietokantaan: pakka on opiskelulista kirjaston yli, ei koskaan sen kopio.</li>
</ul>
<p>Mikään näistä kolmesta ei kirjaa arvosanaa: syrjään siirretty kortti ei ole vastattu kortti, eikä se lasketa istunnon summaan.</p>
<p><strong>Kertausloki.</strong> Pakan asetuksissa <em>Kertausloki</em>-painike näyttää, mitä ajoittajalle <strong>kerrottiin</strong> — päivä, asema, arvosana, tila, myönnetty väli — vastakohtana sille, mitä se suunnittelee. Vain täällä näkee vahingossa annetun arvosanan. Siellä sitä ei voi korjata: aikataulu pysyy ulottumattomissa, ja juuri se sääntö tekee lokista hyödyllisen — menneisyyttä ei voi kirjoittaa uusiksi, mutta sen voi tietää.</p>
<p><strong>Keskeytys/Jatkaminen:</strong> Voit keskeyttää kertausistunnon milloin tahansa näppäimellä <em>Esc</em>. Painike muuttuu muotoon <em>Jatka</em> ja näyttää edistymisesi. Napsauta sitä jatkaaksesi siitä, mihin jäit.</p>
<p><strong>Pakkojen hallinta:</strong> Toimintopainikkeilla voi nimetä uudelleen, synkronoida, nollata tai poistaa pakkoja (kahdesta viimeksi mainitusta pyydetään vahvistus). FSRS-parametrit (tavoitepysyvyys, enimmäisväli, satunnaisuus) asetetaan pakkakohtaisesti Asetuksissa (rattikuvake).</p>
<p><strong>Pysyvyys: tavoite ja mittaus.</strong> <em>Tavoitepysyvyys</em> on sinun valintasi työmäärän ja mieleenpalautuksen laadun välisessä vaihtokaupassa: mitä korkeampi se on, sitä lyhyemmiksi välit käyvät ja sitä enemmän kertaat. Sen rinnalla Asetukset näyttävät <strong>mitatun pysyvyyden</strong> omista kertauksistasi — tieto, ei koskaan ohjaus: blunderDB ei muuta tavoitettasi jahdatakseen onnistumisprosenttiasi. Alle parinkymmenen kertauksen mittausta ei näytetä: se luettaisiin tosiasiaksi, vaikka se on pelkkää kohinaa.</p>
<p>Pysyvyyden muuttaminen <strong>ei vaikuta taannehtivasti</strong>: kukin kortti omaksuu uuden tahdin seuraavassa kertauksessaan, eivätkä jo asetetut eräpäivät siirry. Vaikutus on siis vähittäinen eikä näy samana päivänä.</p>
<p><em>Enimmäisväli</em> rajaa välistyksen. Äskettäin luotu pakka lähtee vuodesta: asema, jonka algoritmi siirtäisi useiden vuosien päähän, on poistunut pakasta ilman että olet niin päättänyt, ja oma pelisi muuttuu sitä nopeammin. Vanhemmat pakat säilyttävät sen arvon, joka niillä oli.</p>
<h3>Harjoittelu-paneeli</h3>
<p><strong>Anki</strong>-paneeli kertaa sitä, mikä <strong>muistetaan</strong>; <strong>Harjoittelu</strong>-paneeli harjoittaa sitä, mikä <strong>lasketaan</strong>, kellon käydessä. Se avautuu näppäimillä <code>CTRL-J</code>, työkalupalkin painikkeesta heti kohdan « Position aléatoire » jälkeen tai komennolla <code>train</code>. Tilannekortti kuuluu molempiin: se lasketaan täällä ja muistetaan tilannekorttien pakassa.</p>
<p>Levossa paneeli näyttää aloittimen ja aiempien istuntojen yhteenvedon.</p>
<h4>Aloitin</h4>
<p>Kolme valintaa, sitten « Démarrer »:</p>
<ul>
<li><strong>harjoitus</strong> — <em>Tulokset</em>, <em>Pipit</em>, <em>Bearoff</em>, <em>Arviointi</em> tai <em>Päätös</em>;</li>
<li>kysymyksen <strong>lähde</strong>, kun harjoituksella on useita — <em>Vivier</em> (harjoituksen vakiomuodot), <em>Plateau</em> (asema sellaisenaan) tai <em>Base</em> (asema selatusta luettelosta);</li>
<li><strong>aikaraja kysymystä kohti</strong> — ei rajaa, 15, 30 tai 60 sekuntia.</li>
</ul>
<p>Valittu lähde muistetaan kullekin harjoitukselle istunnosta toiseen.</p>
<p>Selattava luettelo voi tulla Tilastot-paneelin <em>toistuvista virheistä</em>: napsautus kohtaan "Tietovisa tästä ryhmästä" korvaa sen ryhmän asemilla ja käynnistää Päätös-harjoituksen.</p>
<p><code>train scores</code>, <code>train pips</code>, <code>train bearoff</code>, <code>train evaluation</code> ja <code>train decision</code> avaavat paneelin ja aloittavat suoraan; <code>train tp</code> ja <code>train takepoint</code> ovat <code>train scores</code>:n synonyymejä, <code>train epc</code> on <code>train bearoff</code>:n ja <code>train quiz</code> <code>train decision</code>:n synonyymi.</p>
<h4>Viisi harjoitusta</h4>
<p><strong>Scores</strong> arpoo yhden 36 järjestämättömästä tilanteesta väliltä 2–9 away ja näyttää <strong>tilannekortin</strong>: kaksi saraketta — <em>Vous</em> (sinä) ja <em>L'adversaire</em> (vastustaja) — ja seitsemän riviä — hyväksymispiste kuutiolla 2 ja sitten kuutiolla 4, kumpikin pitkälle kilpajuoksulle ja viimeiselle heitolle, sitten gammonin arvo kuutioilla 1, 2 ja 4.</p>
<p>Ohjerivi muistuttaa kulusta — arvioi jokainen luku päässäsi, paljasta, napsauta ne jotka menivät pieleen — ja lauta näyttää arvotun tilanteen tyhjällä pöydällä.</p>
<p>Kukin sarake kantaa vain ne ruudut, jotka viitetaulukot — ne, jotka komennot <code>tp2_live</code>, <code>tp2_last</code>, <code>tp4_live</code>, <code>tp4_last</code>, <code>gv1</code>, <code>gv2</code> ja <code>gv4</code> näyttävät — määrittelevät sen puolelle: kolme lukua tilanteessa 2a-2a, enintään neljätoista, ja vain yksi sarake tasatilanteessa. Rivi, jota kumpikaan puoli ei määrittele, ei esiinny kortilla — arvattavaa « n/a »-ruutua ei siis ole. Molemmat puolet ovat mukana, koska kuutiopäätös tilanteessa tarvitsee kummankin: korjattu hyväksymispiste yhdistää molempien pelaajien gammonarvot, ja juuri vastustajan hyväksymispiste kertoo, meneekö tuplauksesi läpi.</p>
<p><strong>Pipit</strong> kysyy <strong>molempien</strong> osapuolten pip-luvun. Laudan pip-luku on piilotettu niin kauan kuin kysymys on auki; ”Paljasta” näyttää sen — <strong>vaikka olisit piilottanut pip-luvun</strong> näppäimellä <code>p</code>, sillä muuten vastaus jäisi näkymättömiin eikä harjoitusta voisi tarkistaa. Kyse on peitteestä eikä asetuksesta: omaa valintaasi ei muuteta, ja se palaa voimaan seuraavassa kysymyksessä. Lähde <em>Lauta</em> esittää yhden kysymyksen näytetystä asemasta, ja vain yhden; lähde <em>Tietokanta</em> arpoo uuden aseman joka kysymykseen ja tuo sen laudalle.</p>
<p><strong>Bearoff</strong> kysyy <strong>molempien osapuolten EPC:tä</strong> — tehollista pip-lukua, joka lisää pip-laskentaan niiden nappuloiden hukan, jotka poistuvat yli tarpeen. Tällä alueella moottori on tarkka, ja juuri täällä EPC eroaa todella pip-laskennasta.</p>
<p>Jokainen kysymys <strong>luodaan</strong>: moottori lähtee siemenestä ja pelaa muutaman heiton, ja tilannekuva on kysymys. Satunnainen sijoittelu ei toisi mukanaan oikean ulosvientiaseman aukkoja, matalia pinoja eikä epäsymmetriaa. Siemen tulee <em>Vivier</em>-varannosta (valmis kotiintuonti, sitten nollasta kymmeneen puolisiirtoa), <em>Plateau</em>-laudalta (näytöllä oleva asema, sitten yhdestä neljään puolisiirtoa — ei koskaan nolla, koska olet juuri nähnyt sen) tai <em>Base</em>-kirjastosta (asema selatusta luettelosta sellaisenaan: se on jo todellinen).</p>
<p>Harjoituksen alue: <strong>molemmat osapuolet kokonaan omalla kotialueellaan</strong>, <strong>4–15 nappulaa</strong> puolta kohden ja loput ulos vietyinä, tuplauskuutio keskellä, rahapeli. Siemen, joka ei sovi, <strong>hylätään nimeltä mainiten</strong>, eikä mitään käynnisty — ei hiljaista sopeutusta: pelaaminen kunnes kosketus katkeaa antaisi sinulle aseman, jota et valinnut. Tyhjä lauta on poikkeus: kysymys tulee silloin varannosta, ja paneeli kertoo miksi.</p>
<p>Harjoitus tarvitsee yksipuolisen bearoff-taulukon; niin kauan kuin sitä luodaan taustalla (katso Asetukset), se sanoo sen sen sijaan että esittäisi kysymyksen ilman vastausta.</p>
<p><strong>Évaluation</strong> kysyy, mitä asema on arvoinen: <strong>vuorossa olevan pelaajan voittomahdollisuuden</strong> prosentteina ja <strong>kuutiopäätöksen</strong> — <em>Pas de double</em>, <em>Double, prend</em> tai <em>Double, passe</em>. Ne ovat ne kaksi lukua, jotka Eval-paneeli näyttää, kysyttyinä ennen kuin ne näytetään. Alue on <strong>mikä tahansa asema</strong>: kilpajuoksu yhtä lailla kuin kontaktiasema, <strong>rahapelissä</strong>.</p>
<p>Kuten Bearoffissa, kysymys <strong>luodaan</strong>: siemen tulee kohdasta <em>Vivier</em> (asema, jossa kontakti on juuri katkennut, ja sen jälkeen nollasta kymmeneen moottorin pelaamaa puolisiirtoa), kohdasta <em>Plateau</em> (näytetty asema, ja sen jälkeen yhdestä neljään puolisiirtoa; kysymys esitetään rahapelissä kuutio keskellä, olipa siemenen pistetilanne mikä tahansa) tai kohdasta <em>Base</em> (selattavan luettelon asema sellaisenaan). Tietokannan asema kelpaa vain, jos se on rahapelin kuutiopäätös — ilman noppia, kuutio keskellä tai vuorossa olevalla pelaajalla —; muuten arvonta siirtyy seuraavaan, ja kun mikään ei kelpaa, harjoitus kertoo sen. Lauta, joka ei ole peliasema — ei viittätoista nappulaa kummallakin puolella, tai päättynyt peli — <strong>hylätään syy kertoen</strong>; tyhjällä laudalla kysymys tulee varannosta.</p>
<p>Totuus on moottorin: kaksipuolinen bearoff-tietokanta, kun asema on siinä, muualla gammonNet kanonisella syvyydellään, ja paneeli kertoo, kumpi vastasi. Mitään ei arvioida karkeasti: asemaa, jota moottori ei arvioi, ei kysytä.</p>
<p><strong>Décision</strong> esittää <strong>jo analysoidun</strong> päätöksen: selattavan luettelon aseman — sen ainoan lähteen — ja päätöksen, jota asema kantaa, nappulasiirron tai kuutiopäätöksen, ja tallennettu analyysi arvioi. Asema ilman analyysiä ei esitä kysymystä, eikä kerran kysytty asema palaa istunnossa; kun luettelo on käyty läpi, paneeli kertoo sen. Ilman avointa tietokantaa tai ilman analysoitua asemaa luettelossa harjoitus <strong>kieltäytyy ja kertoo syyn</strong>, eikä mitään aloiteta.</p>
<p>Kun <em>Évaluation</em>- tai <em>Décision</em>-kysymys odottaa vastaustaan, analyysipaneeli pysyy peitettynä: se kantaa vastauksen.</p>
<h4>Vastaaminen</h4>
<p>Vastaustapa on harjoituksen ominaisuus, ei koskaan asetus: se mitä <strong>lasketaan tai muistetaan</strong> paljastetaan, se mitä <strong>arvioidaan</strong> kirjoitetaan — sillä siellä virheen suuruus on opetus.</p>
<p><em>Tulokset</em> ja <em>Pipit</em> <strong>ilmoitetaan</strong>: lasket päässäsi, napsautat ”Paljasta”, ja totuus ilmestyy. Jokainen luku on silloin <strong>oletuksena oikein</strong> — napsautat sitä, jonka menit väärin, merkitäksesi sen <strong>virheeksi</strong> (<em>Sarkain</em> ja sitten <em>Välilyönti</em> tekee saman näppäimistöltä), ja toinen napsautus poistaa merkinnän. Mitään ei kirjoiteta: pip-luku tai taulukon solu on oikein tai väärin, eikä sen kirjoittaminen opeta enempää kuin sen lukeminen.</p>
<p><em>Bearoff</em> <strong>kirjoitetaan</strong>: kirjoitat molemmat EPC-luvut, « Valider » arvioi ne puolen pipin tarkkuudella — se on tarkkuus, jolla EPC muuttaa kilpajuoksun ratkaisun — ja totuus ilmestyy kirjoittamasi viereen. Sovellus arvioi, mitään ei tarvitse rastittaa. Poikkeama kirjataan <strong>etumerkkeineen</strong>: yliarviointi ei ole aliarviointia, ja yhteenveto tekee siitä keskiarvon.</p>
<p><em>Évaluation</em> yhdistää molemmat eleet samaan kysymykseen. Voittomahdollisuus <strong>kirjoitetaan</strong> ja arvioidaan viiden prosenttiyksikön tarkkuudella, etumerkillinen poikkeama mukaan lukien; kuutiopäätös <strong>valitaan</strong> — napsautus pitää painikkeen arvioimatta mitään, ja « Valider » arvioi molemmat kerralla. Kuutiolla ei ole toleranssia: oikein on vain se painike, jonka moottorin tuomio tekee oikeaksi, ja asemaan, joka on <em>liian hyvä tuplaamiseen</em>, vastataan <em>Pas de double</em>. <em>Enter</em> kentässä vie kuutiopäätökseen niin kauan kuin sitä ei ole valittu, ja hyväksyy sen jälkeen. Vastauksen jälkeen paneeli näyttää totuuden — neljän lopputuloksen tuomion —, sen lähteen ja <strong>molempien osapuolten EPC:n</strong>, kun asemalla on tarkka EPC; EPC:tä ei koskaan kysytä tässä, sillä on oma harjoituksensa. Päiväkirja laskee kaksi lukua erikseen: aseman voi arvioida hyvin ja lukea sen kuution väärin.</p>
<p><em>Décision</em> <strong>valitaan</strong>. Nappulasiirrossa <strong>pelaa siirto laudalla</strong>: napsauta lähtöpistettä ja sitten kohdetta tai vedä nappula, kerran kutakin noppaa kohti. Lauta tarjoaa vain sen, mitä voi pelata — napsautus, jota mikään laillinen siirto ei salli, ei liikuta mitään. Paneelissa « Annuler le pas » peruu yhden nopan, « Recommencer » palauttaa aseman sellaisena kuin kysymys sen esittää (kaksoisnapsautus laudan ulkopuolella tekee saman), ja « Valider », joka on käytössä kun siirto on valmis, lähettää sen arvioitavaksi. Merkintäkenttä hyväksyy myös kirjoitetun siirron (<code>13/7 8/7</code>, litteroinnin merkintätapa): askeleet asetetaan laudalle jokaisella näppäilyllä, punaiseksi muuttunut kenttä kertoo, ettei jokin askel ole pelattavissa, ja ENTER vahvistaa koko siirron. Kun paneelilla on fokus, ASKELPALAUTIN kumoaa yhden askeleen, ESC aloittaa siirron alusta ja ENTER vahvistaa sen. Kuutiopäätöksessä napsauta <em>Ei tuplausta</em>, <em>Tuplaus, hyväksy</em> tai <em>Tuplaus, luovuta</em>: napsautus on vastaus.</p>
<p>Korjaus pitää kolme lopputulosta erillään, ja niiden sekoittaminen valehtelisi. <strong>Sääntöjenvastainen siirto</strong> ei ole huonosti valittu siirto — se on sääntövirhe. <strong>Laillinen siirto, jota moottori ei arvioinut</strong>, ei ole arviointivirhe: sillä ei yksinkertaisesti ole hintaa, eikä se maksa mitään. Arvioitu siirto maksaa sen, minkä analyysi sanoo, millipisteinä. Oikein on vain arvioitu siirto, joka ei maksa mitään; paras siirto näytetään joka tapauksessa.</p>
<p>Kello käynnistyy kysymyksen ilmestyessä ja pysähtyy kohdassa « Révéler », « Valider » tai <em>Décision</em>-harjoituksen kuutiopäätöksen napsautuksessa; seuraava kysymys valmistellaan sillä aikaa kun vastaat, joten sitä ei koskaan mitata sinun kanssasi. Virheiden rastittamista ei myöskään mitata. Rajan kanssa määräajassa vastaamatta jäänyt kysymys paljastuu itsestään ja lasketaan <strong>myöhästyneeksi</strong>: kaikki sen luvut ovat väärin, eikä sen aika mene mediaaniin — vastausta jota ei annettu ei mitata. Myöhästynyt päätös näyttää parhaan siirron, eikä se mene istunnon PR:ään.</p>
<p>« Suivante » tallentaa kysymyksen ja esittää uuden. Istunnolla ei ole kiinteää pituutta: se kestää kunnes « Terminer », joka kirjoittaa sen päiväkirjaan, tai « Quitter », joka hylkää sen. Kaikki painikkeet ovat paneelissa; lauta näyttää kysymyksen ja sen vastauksen, se ei kanna yhtään säädintä.</p>
<p>Kun kysymys on esitetty laudalla, paljastettuna tai ei, luetteloa selaavat näppäimet eivät vieritä sitä: kysymys pitää laudan, kunnes valitaan « Suivante », « Terminer » tai « Quitter ».</p>
<h4>Päiväkirja ja yhteenveto</h4>
<p>Päättyneet istunnot säilyvät itse tietokannassa — ne siis seuraavat tiedostoa — eikä niillä ole ylärajaa. Levossa paneeli näyttää yhden rivin harjoitusta kohti: istuntojen määrän, virheprosentin, mediaaniajan ja, kymmenestä istunnosta alkaen, <strong>suuntauksen</strong>, eli eron kymmenen viimeisen istunnon virheprosentin ja kaikkien istuntojen virheprosentin välillä — negatiivisena edistyt.</p>
<p><em>Décision</em>-harjoituksessa rivi antaa myös viimeisen istunnon <strong>PR:n</strong>, joka lasketaan samalla kaavalla kuin tilastot laskevat oikealle pelille — 500 × keskimääräinen virhe normalisoituna ekviteettinä arvioiduista päätöksistä. Harjoittelun PR 6 ja ottelun PR 6 mittaavat samaa asiaa samalla asteikolla.</p>
<p>Harjoituksen nimeä napsauttamalla avautuu erittely <strong>lukutyypeittäin</strong>: « Point de prise 4 · dernier lancer, 6 / 9 ». Juuri tämä erittely tekee päiväkirjasta hyödyllisen, ja se laskee tyypin eikä puolen mukaan: saman taulukon sama ruutu, kummalta puolelta tahansa katsottuna, on yksi ja sama heikkous.</p>
<p><em>Päätös</em>-kysymys säilyttää päiväkirjassa asemansa, annetun vastauksen ja sen hinnan millipisteinä. <em>Päätös</em>-harjoituksen tiedot sisältävät siksi kolme painiketta, jotka toimivat kaikilla epäonnistuneilla asemilla, kukin kerran, viimeksi epäonnistunut ensin — aikarajan ylittänyt kysymys lasketaan epäonnistuneeksi:</p>
<ul>
<li><strong>Kertaa virheeni</strong> — epäonnistuneista asemista tulee selattava lista, ja <em>Päätös</em>-istunto alkaa uudelleen niillä;</li>
<li><strong>Virheiden Anki-pakka</strong> — Anki-pakka näistä asemista;</li>
<li><strong>Virheiden kokoelma</strong> — kokoelma näistä asemista.</li>
</ul>
<p>Pakan ja kokoelman nimi on ”Päätöksen virheet” ja perässä päivän päivämäärä. Komentorivillä <code>training missed</code> antaa saman listan ja tekee siitä pakan (<code>--deck</code>) tai kokoelman (<code>--collection</code>), ja <code>training sessions</code> lukee päiväkirjan uudelleen (katso training — Harjoituspäiväkirja).</p>
<h3>Metatietopaneeli</h3>
<p><strong>Metatiedot</strong>-paneeli (<em>CTRL-M</em>) näyttää nykyisen tietokannan yleistiedot: <em>Käyttäjä</em>, luontipäivä (<em>Luotu</em>), skeeman <em>Versio</em> ja <em>Kuvaus</em>. Käyttäjä, päivämäärä ja kuvaus muokataan paikan päällä ja tallennetaan kentästä poistuttaessa; versio on vain luettavissa. Saatavilla myös komennolla <code>meta</code>.</p>
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
<p>Kentät <em>Käyttäjä</em>, <em>Kuvaus</em> ja <em>Luontipäivä</em> kuvaavat <strong>syntyvää tiedostoa</strong>; ne on esitäytetty lähdetietokannasta. Valintaruutu <em>Omat tallennetut suodattimet</em> on erillinen muista: se ei vie sisältöä vaan omat tallennetut hakusi, joista ei ole hyötyä jonkun toisen tietokannassa.</p>
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
<li>”✓ sinun merkitsemäsi”: tiedostossa on oma merkintäsi ehjänä;</li>
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
<p>Työkalupalkin vihjeet muistuttavat kunkin painikkeen näppäimestä sillä nimellä, joka sillä on käyttöliittymän kielen näppäimistössä: <em>Vasen</em>, <em>Poista</em>, <em>Page Up</em>, <em>Page Down</em>, <em>Vaihto</em>. Alla olevat taulukot ja ohjeikkuna käyttävät samoja nimiä.</p>
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
<td>CTRL-VAIHTO-I</td>
<td>Yhdistä tietokanta tähän.</td>
</tr>
<tr>
<td>CTRL-VAIHTO-S</td>
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
<td>CTRL-VAIHTO-F</td>
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
<td>Muokkaus- tai Eval-tilassa: nollaa lauta, kuutio, tulos ja nopat.</td>
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
<td>Home, h</td>
<td>Ensimmäinen asema / Edellinen peli (ottelunavigointi).</td>
</tr>
<tr>
<td>Page Up</td>
<td>Siirtyy sivun taaksepäin (oletuksena sata asemaa, säädettävissä kohdassa Asetukset &gt; Käyttöliittymä: 10, 50, 100, 500 tai 1 000 asemaa tai 10 % luettelosta; pysähtyy luettelon alkuun); ottelussa edellinen peli.</td>
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
<td>End, l</td>
<td>Viimeinen asema / Seuraava peli (ottelunavigointi).</td>
</tr>
<tr>
<td>Page Down</td>
<td>Siirtyy sivun eteenpäin (sama askel kuin <em>Page Up</em>; pysähtyy luettelon loppuun); ottelussa seuraava peli.</td>
</tr>
<tr>
<td>r</td>
<td>Lataa satunnainen asema.</td>
</tr>
<tr>
<td>ESC</td>
<td>Poistu kokoelmasta tai ottelusta käynnistetyn <code>ss</code>-haun tuloksista: paluu kokoelmaan tai ottelun tarkasteltuun siirtoon.</td>
</tr>
</tbody>
</table>
<p>Kun Harjoittelu-paneelin kysymys on esitetty laudalla, luetteloa selaavat näppäimet eivät vieritä sitä: kysymys pitää laudan. Nappulasiirrossa, kun paneelilla on fokus: ASKELPALAUTIN kumoaa viimeisen askeleen, ESC aloittaa siirron alusta, ENTER vahvistaa sen, kun se on valmis; merkintäkentässä ENTER vahvistaa kirjoitetun siirron (<code>13/7 8/7</code>).</p>
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
<tr>
<td>ALT-1 … ALT-9</td>
<td>Käynnistä suodatinkirjaston kyseisen sijan kiinnitetty suodatin.</td>
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
<td>CTRL-VAIHTO-L</td>
<td>Järjestä nykyisen aseman naapurit.</td>
</tr>
<tr>
<td>CTRL-P</td>
<td>Näytä/piilota kommentit.</td>
</tr>
<tr>
<td>CTRL-VAIHTO-P</td>
<td>Avaa/sulje komentopaletti: komennot, välilehdet, suodattimet ja ottelut likimääräisellä nimellä.</td>
</tr>
<tr>
<td>CTRL-J</td>
<td>Näytä/piilota Harjoittelu-paneeli.</td>
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
<td>CTRL-VAIHTO-T</td>
<td>Näytä/piilota Litterointi-paneeli (otteluluonnokset).</td>
</tr>
<tr>
<td>J / K</td>
<td>Johdetun turnauksen ehdotuslistassa: alas, ylös.</td>
</tr>
<tr>
<td>ENTER</td>
<td>Ehdotuslistassa: vahvista valittu ehdotus.</td>
</tr>
<tr>
<td>Turnaussivu (TAB)</td>
<td>Ensimmäinen pysäkki Johto-välilehdellä: ”Siirry jonoon”; ENTER siirtää kohdistuksen ehdotusjonoon.</td>
</tr>
<tr>
<td>VASEN / OIKEA</td>
<td>Tuloskortissa kentän ulkopuolella: valitse vasemmanpuoleinen tai oikeanpuoleinen pelaaja; ENTER kirjaa hänen voittonsa.</td>
</tr>
<tr>
<td>CTRL-Z</td>
<td>Peru johdetun turnauksen viimeisin päätös (syöttökentän ulkopuolella, jossa se kumoaa näppäilyn).</td>
</tr>
<tr>
<td>TAB (kohdistus kadonnut)</td>
<td>Johdetun turnauksen sivulla, kun kohdistus on pudonnut sivulle: palauttaa sen sivun ensimmäiseen elementtiin avaamatta hakua.</td>
</tr>
<tr>
<td>PAGE UP / PAGE DOWN, HOME / END</td>
<td>Johdetun turnauksen sivun alla: vieritä sivua selaamatta sen peittämää lautaa.</td>
</tr>
<tr>
<td>ESC</td>
<td>Sulje tuloskortti tai käynnissä oleva peruutus.</td>
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
<td>CTRL-Page Up, VAIHTO-J</td>
<td>Edellinen näkymä.</td>
</tr>
<tr>
<td>CTRL-Page Down, VAIHTO-K</td>
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
<tr>
<td>ESC</td>
<td>Haun aikana: keskeyttää haun.</td>
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
<tr>
<td>Napsautus (tähteen)</td>
<td>Kiinnitä suodatin tai irrota se.</td>
</tr>
<tr>
<td>Napsautus (kiinnitettyyn merkkiin)</td>
<td>Suorita kiinnitetyn suodattimen haku.</td>
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
<td>r</td>
<td>Käynnistä aseman rollout valitulla asetuksella; toinen painallus pysäyttää sen.</td>
</tr>
<tr>
<td>Esc</td>
<td>Poista siirron valinta. Jos mitään siirtoa ei ole valittu, sulje paneeli, paitsi <code>ss</code>-haun tulosten edessä: palaa niistä kokoelmaan tai otteluun.</td>
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
<td>/</td>
<td>Siirry suodatuskenttään (pelaaja, tapahtuma, turnaus, päivämäärä). <em>Esc</em> tyhjentää suodattimen.</td>
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
<td>Kaksoisnapsautus, ENTER (kohdistettu rivi)</td>
<td>Valitse turnaus (näytä sen tiedot). TAB tavoittaa rivit; yksittäinen napsautus vain korostaa.</td>
</tr>
<tr>
<td>YLÖS, k</td>
<td>Valitse edellinen turnaus, kun paneelilla on kohdistus tai yhtään johdettua turnausta ei ole näkyvissä.</td>
</tr>
<tr>
<td>ALAS, j</td>
<td>Valitse seuraava turnaus, kun paneelilla on kohdistus tai yhtään johdettua turnausta ei ole näkyvissä.</td>
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
<h3>Johtamissivu</h3>
<p>Direction-sivulla <em>J</em>, <em>K</em>, <em>YLÖS</em>, <em>ALAS</em> ja <em>ENTER</em> siirtyvät ehdotusjonoon, paitsi kun kohdistus on pöytäruudukon ruudussa, jossa <em>YLÖS</em> ja <em>ALAS</em> vaihtavat ruutua. Kontekstivalikot on kuvattu käsikirjassa (kontekstivalikot).</p>
<table>
<thead>
<tr>
<th>Oikotie</th>
<th>Toiminto</th>
</tr>
</thead>
<tbody>
<tr>
<td>Clic droit, MENU, VAIHTO-F10</td>
<td>Avaa kohdistetun kohteen kontekstivalikko: pöytäruutu, pelaaja, kaavion paikka, paikka, ehdotus, historian rivi. YLÖS/ALAS liikkuvat valikossa, ENTER valitsee, ESC sulkee sen.</td>
</tr>
<tr>
<td>VASEN, OIKEA, YLÖS, ALAS, HOME, END</td>
<td>Siirry pöytäruudukon ruudusta toiseen (vapaat ruudut mukaan lukien); ruudukko vie vain yhden TAB-pysähdyksen.</td>
</tr>
<tr>
<td>1–9, sitten 0–9</td>
<td>Avaa kyseisen numeron pöydän kortti; kaksi numeroa 0,4 sekunnin kuluessa yli 9:n pöydälle.</td>
</tr>
<tr>
<td>M, X</td>
<td>Varatussa ruudussa avaa kortti (tai, jos se on auki, pöytäkenttä); varattuun pöytään osoittaminen vaihtaa kaksi ottelua keskenään.</td>
</tr>
<tr>
<td>/</td>
<td>Avaa pikahaku: tapahtuman pelaajat, pöydät, käynnissä olevat ottelut ja kilpailut löytyvät nimellä tai pöydän numerolla (lisätiedot). Ei vaikutusta syöttökentässä.</td>
</tr>
<tr>
<td>F11</td>
<td>Aseta turnauksenjohdon sivu koko näytön tilaan (työkalupalkki, välilehdet, paneeli ja tilapalkki piilotettuina) tai poistu siitä. ESC poistuu myös, kun avoin valikko tai kortti on suljettu (lisätiedot).</td>
</tr>
<tr>
<td>Vedä varattu ruutu toisen päälle</td>
<td>Hiirellä: vapaaseen ruutuun siirretään ottelu; varattuun ruutuun kaksi ottelua vaihdetaan vahvistuksen jälkeen. ESC peruu vedon.</td>
</tr>
<tr>
<td>Kaikki pöydät</td>
<td>Tapahtuman <em>Kaikki pöydät</em> -ruudukossa samat näppäimet, valikot ja vetäminen vaikuttavat minkä tahansa kilpailun pöytään; vaihto toisen kilpailun kanssa mainitsee vahvistuksessa molemmat.</td>
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
<h3>Litterointipaneeli</h3>
<p>Paneeli ottaa nämä näppäimet kun sillä on kohdistus. Luonnosluettelon edessä se ottaa jo muutaman, jotta eilisen työn jatkaminen ei vaatisi hiirtä.</p>
<table>
<thead>
<tr>
<th>Oikotie</th>
<th>Toiminto</th>
</tr>
</thead>
<tbody>
<tr>
<td>ALAS, j / YLÖS, k (luonnosluettelo)</td>
<td>Selaa luonnoksia. Ensimmäinen on korostettuna avattaessa: se on viimeksi muutettu.</td>
</tr>
<tr>
<td>ENTER (luonnosluettelo)</td>
<td>Avaa korostettu luonnos.</td>
</tr>
<tr>
<td>n (luonnosluettelo)</td>
<td>Avaa luontilomake.</td>
</tr>
<tr>
<td>Napsautus</td>
<td>Avaa luonnos luettelosta.</td>
</tr>
<tr>
<td>1 … 6</td>
<td>Syötä noppa. Pelin ensimmäisessä siirrossa: pelaajan 1 noppa, sitten pelaajan 2 (suurempi aloittaa ja pelaa molemmat nopat).</td>
</tr>
<tr>
<td>1 … 6 (heitto syötetty)</td>
<td>Vahvista valittu siirto ja avaa seuraava heitto. Uudelleen tarkasteltavassa toiminnossa, jossa kohdistin on jo kirjoitetun toiminnon päällä, numero aloittaa tämän toiminnon heiton uudelleen vahvistamisen sijaan.</td>
</tr>
<tr>
<td>1 … 6 (kohdistin pelin ensimmäisen siirron kohdalla)</td>
<td>Kirjoita toinen avausheitto: pelaajan 1 noppa, sitten pelaajan 2. Suurempi voittaa — iso noppa ensin, siirto pelaajalle 1 alhaalla; pieni noppa ensin, pelaajalle 2 ylhäällä; ensimmäinen ehdokas esivalitaan. Sama heitto uudelleen ei muuta mitään; <em>s</em> antaa siirron toiselle pelaajalle.</td>
</tr>
<tr>
<td>ALAS, j</td>
<td>Valitse seuraava ehdokas (siirron nuolet ilmestyvät laudalle).</td>
</tr>
<tr>
<td>YLÖS, k</td>
<td>Valitse edellinen ehdokas.</td>
</tr>
<tr>
<td>Rulla</td>
<td>Valitse seuraava tai edellinen ehdokas sekä luettelon että laudan päällä: katse pysyy laudalla ja nuolet vierivät ohi.</td>
</tr>
<tr>
<td>Napsautus (riviä)</td>
<td>Valitse kyseinen ehdokas.</td>
</tr>
<tr>
<td>Kaksoisnapsautus (riviin)</td>
<td>Vahvista tämä ehdokas.</td>
</tr>
<tr>
<td>Napsautus (heittokolmiossa)</td>
<td>Syötä heitto yhdellä eleellä: ruutu sisältää molemmat nopat, tuplat lävistäjällä. Pelin ensimmäisessä siirrossa kolmio väistyy kuuden nopan rivin tieltä, ja yksi napsautus antaa yhden puolen nopan.</td>
</tr>
<tr>
<td>Napsautus, vetäminen (ei syötettyä noppaa)</td>
<td>Pelaa siirto suoraan laudalla: nappula kulkee napsautetusta pisteestä kohteeseensa laillisten siirtojen rajoissa, ja molemmat nopat päätellään pelatuista askelista.</td>
</tr>
<tr>
<td>Napsautus, vetäminen (heitto syötetty)</td>
<td>Pelaa siirto laudalla tämän heiton laillisten siirtojen rajoissa: jokainen pelattu askel jättää listaan vain sen sisältävät ehdokkaat, ja valmis laillinen siirto kirjataan heti. Uudelleen tarkasteltavassa toiminnossa se korvaa toiminnon.</td>
</tr>
<tr>
<td>Vetäminen sääntöjen ulkopuolelle (heitto syötetty)</td>
<td>Laske nappula siihen, mihin se päästetään, myös pisteestä, josta ei lähde yhtään laillista siirtoa, laittoman siirron kirjaamiseksi. Siirron loppu pelataan vapaasti, napsauttamalla tai vetämällä, ja ehdokaslista väistyy rivin tieltä, joka muistuttaa siitä.</td>
</tr>
<tr>
<td>ENTER (sääntöjen vastainen siirto)</td>
<td>Kirjaa siirto syötetyillä nopilla ja saadulla laudalla, laittomaksi siirroksi merkittynä, jos mikään laillinen siirto ei johda tähän lautaan. Sääntöjen vastaista siirtoa ei koskaan kirjata itsestään.</td>
</tr>
<tr>
<td>ASKELPALAUTIN (siirto kesken laudalla)</td>
<td>Kumoa viimeinen laudalla pelattu askel. Jäljelle jäävät askeleet pelataan uudelleen rajoitettuina niin kauan kuin jokin laillinen siirto sisältää ne: ainoan sääntöjen vastaisen askeleen kumoaminen palauttaa listan.</td>
</tr>
<tr>
<td>ENTER</td>
<td>Kirjaa valittu siirto (pelin viimeinen siirto).</td>
</tr>
<tr>
<td>ASKELPALAUTIN</td>
<td>Pyyhi kaksi syötettyä noppaa. Sitä kautta kulkee väärin luetun heiton uusiminen, koska numero vahvistaa.</td>
</tr>
<tr>
<td>Napsautus (heiton ruutuihin)</td>
<td>Pyyhi kaksi syötettyä noppaa, kuten ASKELPALAUTIN.</td>
</tr>
<tr>
<td>Esc</td>
<td>Hylkää kesken oleva syöttö.</td>
</tr>
<tr>
<td>d</td>
<td>Tuplaa tai uudelleentuplaa: valittu siirto vahvistetaan samalla, yhdellä ainoalla näppäimellä.</td>
</tr>
<tr>
<td>t</td>
<td>Ota tarjottu tuplaus: kuutio siirtyy ottajalle kaksinkertaiseen arvoon ja tuplaaja heittää uudelleen. Kun kohdistin on solussa, kirjoittaa oton kohdennetun toiminnon tilalle; otoksi muuttunut luovutus avaa pelinsä uudelleen, ja loput kirjoitetaan sen sijaan siihen.</td>
</tr>
<tr>
<td>p</td>
<td>Luovuta tarjotun tuplauksen edessä: peli voitetaan sillä arvolla, joka kuutiolla oli ennen tuplausta. Kun kohdistin on solussa, kirjoittaa luovutuksen kohdennetun toiminnon tilalle.</td>
</tr>
<tr>
<td>r ja sitten 1, 2 tai 3</td>
<td>Luovuta peli vuorossa olevan puolesta: yksinkertainen, gammon tai backgammon. Esc näppäinten välissä peruuttaa tallentamatta mitään.</td>
</tr>
<tr>
<td>Napsautus (tuplauskuutiota)</td>
<td>Tarjoa tuplausta vuorossa olevalle puolelle, kuten d-näppäin. Tarjoukseen kuutio ei vastaa: vastaanotto ja passaus ovat painikerivillä.</td>
</tr>
<tr>
<td>VASEN, h</td>
<td>Siirrä kohdistinta yksi toiminto taaksepäin siirtoluettelossa.</td>
</tr>
<tr>
<td>OIKEA, l</td>
<td>Siirrä kohdistinta yksi toiminto eteenpäin.</td>
</tr>
<tr>
<td>Napsautus (solua)</td>
<td>Vie kohdistin tähän toimintoon tai kaksoisvuoron puuttuvaan vuoroon.</td>
</tr>
<tr>
<td>Kaksoisnapsautus (solua)</td>
<td>Kirjoita tämän toiminnon siirto näppäimistöllä, solun sisään: 13/7 8/7*, bar/22, 6/off. Vain siirto kirjoitetaan, nopat ovat solun omat; ENTER kirjaa sen, laittomankin, ja Esc sulkee solun kirjoittamatta mitään. Koskee siirtoa, tanssia, kirjaamatonta siirtoa sekä kesken olevan syötön katkoviivasolua heti kun sen molemmat nopat on syötetty.</td>
</tr>
<tr>
<td>Kaksoisnapsautus (pelin tilannetta)</td>
<td>Kirjoita tilanne, jolla tämä peli pelattiin, sen otsikkoon: 3-2, 3–2 tai 3 2. ENTER kirjaa sen, tyhjennetty kenttä palaa tilanteeseen, jonka aiemmat pelit antavat, ja Esc sulkee kentän kirjoittamatta mitään. Siitä poikkeava tilanne merkitään epäjohdonmukaisuudeksi. Rahapelissä ei tilannetta.</td>
</tr>
<tr>
<td>Napsautus hiiren oikealla (solua)</td>
<td>Avaa kyseisen toiminnon korjaukset: lisää ennen, lisää jälkeen, poista, vaihda puolta. Kohdistin tuodaan solun päälle samalla.</td>
</tr>
<tr>
<td>CTRL-ENTER</td>
<td>Viimeistele luonnos: kirjoita ottelu tai korvaa se, josta luonnos avattiin, ja vapauta luonnos.</td>
</tr>
<tr>
<td>i</td>
<td>Lisää toiminto kohdistimen toiminnon eteen (ehdotettu puoli on se, joka pitää jatkon johdonmukaisena).</td>
</tr>
<tr>
<td>a</td>
<td>Lisää toiminto kohdistimen toiminnon jälkeen.</td>
</tr>
<tr>
<td>x, Del</td>
<td>Poista muokattavana oleva päätös — kohdistimen toiminto tai vielä kirjoittamaton syöttö — ja palaa edelliseen, valmiina korjattavaksi; seuraavat säilyttävät puolensa. Asiakirjan lopussa palaa viimeiseen toimintoon.</td>
</tr>
<tr>
<td>s</td>
<td>Anna kohdistimen toiminto toiselle puolelle.</td>
</tr>
<tr>
<td>CTRL-Z</td>
<td>Kumoa viimeisin luonnokseen tehty ele.</td>
</tr>
<tr>
<td>CTRL-VAIHTO-Z</td>
<td>Tee kumottu ele uudelleen.</td>
</tr>
</tbody>
</table>
<p>Heitto, joka ei salli yhtään siirtoa, kirjaa tanssin itsestään ilman lisänäppäintä.</p>
<p>Numerolla on yksi ainoa merkitys: <strong>se aloittaa heiton siellä missä kohdistin on</strong>. Asiakirjan lopussa kohdistimen alla ei ole mitään, joten se vahvistaa valitun siirron ennen seuraavan heiton avaamista — parhaan pelatun siirron hinnaksi tulee näin kaksi noppaa eikä enempää, sillä sen vahvistuksen kantaa seuraavan vuoron ensimmäinen näppäin. Jo kirjoitetussa toiminnossa, jonka luo on palattu korjaamaan, kohdistimen alla on jotakin: numero aloittaa tämän toiminnon heiton uudelleen, paikallaan. Ero näkyy näytöllä, sillä kohdennettu solu on kehystetty transkriptissä. Asiakirjan <strong>viimeinen</strong> toiminto on poikkeus: kun sen heitto on kirjoitettu uudelleen, seuraava numero vahvistaa sen ja avaa seuraavan päätöksen, kuten asiakirjan lopussa, ja ENTER vie myös sinne.</p>
<p>Se mitä parhaillaan kirjoitetaan piirtyy transkriptiin katkoviivalla siihen paikkaan, johon se kirjataan: korjaus peittää solun, jonka se korvaa, lisäys avaa solun kahden naapurinsa väliin, ja puoli luetaan sarakkeesta. Mitään ei kirjata ennen vahvistusta.</p>
<p>Keskelle asiakirjaa tehty lisäys jatkaa lisäämistä: vahvistus avaa tyhjän solun sen perään, ja seuraava toiminto lisätään vuorostaan sen sijaan että se korvaisi jälkimmäisen. Pelin loppu tai kohdistimen siirtäminen päättää sen.</p>
<p>Ele, jolla ei ole mitään tehtävää, sanoo sen kerran tilapalkissa: ”ei mitään kumottavaa” tyhjällä pinolla, ”ei toimintoa kohdistimen alla” asiakirjan lopussa. Lause pyyhkiytyy itsestään ja antaa paikan takaisin odotetulle toiminnolle.</p>
<p>Kohdistimen siirtäminen takaisin toiminnon kohdalle ja uudelleen kirjoittaminen korjaa sen <strong>paikallaan</strong>: hyväksyntä korvaa toiminnon ja kohdistin palaa sinne, missä se oli — tai viimeisen toiminnon kohdalla siirtyy asiakirjan loppuun, jossa litterointi jatkuu. Jos nopat korjataan ja tallennettu siirto on yhä uuden heiton laillinen siirto, se säilytetään; muuten tarjotaan uuden heiton ensimmäistä ehdokasta ja siirto merkitään ”tarkistettavaksi” hyväksyntään asti. Kohdistimen siirtäminen eteen- tai taaksepäin muutoksen jälkeen tallentaa korjauksen samalla.</p>
<p>Mitään ei hylätä eikä poisteta: naapurinsa kanssa samalle puolelle lisätty toiminto luo kaksoisvuoron, toiminnon poistaminen voi luoda toisen, puolen vaihtaminen voi tehdä seuraavista siirroista laittomia. Nämä epäjohdonmukaisuudet merkitään transkriptiin, niitä ei koskaan korjata automaattisesti, ja kohdistin asettuu ensimmäisen niistä kohdalle jokaisen kirjoittavan eleen jälkeen — paitsi siellä, missä jatketaan: poiston, lisäyksen tai uudelleen avatun pelin jälkeen se pysyy siinä, mihin kirjoitetaan. Liikkuminen tai nopan kirjoittaminen ei koskaan siirrä sitä. Vuoro, jonka kaksoisvuoro on kadottanut, on transkriptin solu: h ja l pysähtyvät siihen, ja siihen kirjoitettu heitto lisätään sille puolelle, jolta se puuttui. Kumoamispino elää muistissa: se menetetään, kun luonnos suljetaan.</p>
<p>Paneeli itse — luonnosten luettelo, luonti, syöttö, siirtoluettelo ja luonnospalkki — kuvataan kohdassa Litterointipaneeli.</p>
<h3>Vedosarkki</h3>
<table>
<thead>
<tr>
<th>Oikotie</th>
<th>Toiminto</th>
</tr>
</thead>
<tbody>
<tr>
<td>VASEN, OIKEA, YLÖS, ALAS</td>
<td>Liiku ruudukossa.</td>
</tr>
<tr>
<td>j, k</td>
<td>Seuraava pienoiskuva, edellinen pienoiskuva.</td>
</tr>
<tr>
<td>Home, End</td>
<td>Sivun ensimmäinen, viimeinen pienoiskuva.</td>
</tr>
<tr>
<td>Page Up, Page Down</td>
<td>Edellinen sivu, seuraava sivu.</td>
</tr>
<tr>
<td>ENTER, Napsautus</td>
<td>Avaa asema laudalle ja sulje sarkki.</td>
</tr>
<tr>
<td>Esc</td>
<td>Sulje sarkki.</td>
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
<td>Page Up</td>
<td>Sisällön alkuun.</td>
</tr>
<tr>
<td>Page Down</td>
<td>Sisällön loppuun.</td>
</tr>
<tr>
<td>/</td>
<td>Hae ohjeesta: Enter siirtyy seuraavaan osumaan, SHIFT-Enter edelliseen.</td>
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
<td>eval, epc</td>
<td>Avaa Eval-paneelin (Effective Pip Count, voittotodennäköisyys ja kuutiopäätös bearoffissa). <code>epc</code> on tämän paneelin vanha nimi, joka on säilytetty.</td>
</tr>
<tr>
<td>transcribe, tr</td>
<td>Avaa Litterointi-paneelin: kirjattavana olevat otteluluonnokset ja keinon aloittaa uusi.</td>
</tr>
<tr>
<td>direct</td>
<td>Avaa Turnaukset-paneelin turnauksen johtamista varten ja sulkee sen taas. Kun siellä on johtaminen auki, pääalue näyttää turnauksen laudan sijaan; mikä tahansa muu välilehti tuo laudan takaisin.</td>
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
<td>train</td>
<td>Avaa Harjoittelu-paneelin. Argumentin kanssa se avaa ja aloittaa: <code>train scores</code> (arvotun tilanteen pistekortti; <code>train tp</code> ja <code>train takepoint</code> ovat synonyymejä), <code>train pips</code> (molempien osapuolten pip-laskenta), <code>train bearoff</code> (molempien osapuolten EPC luodussa asemassa; <code>train epc</code> on synonyymi), <code>train evaluation</code> (luodun aseman voittomahdollisuus ja kuutiopäätös, rahapelissä), <code>train decision</code> (analysoitu päätös selattavasta luettelosta: siirto pelataan laudalla, kuutiopäätös valitaan paneelissa; <code>train quiz</code> on synonyymi).</td>
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
<td>resume</td>
<td>Luettelee tuonnit, joita mikään ei ole päättänyt (sovellus pysäytettiin) ja jatkaa valittua, kun kansio tai tiedostot on osoitettu uudelleen.</td>
</tr>
<tr>
<td>[number]</td>
<td>Siirry annetun indeksin asemaan.</td>
</tr>
<tr>
<td>[number]%</td>
<td>Siirtyy luettelon tähän prosenttikohtaan: <code>0%</code> ensimmäinen asema, <code>50%</code> keskikohta, <code>100%</code> viimeinen.</td>
</tr>
<tr>
<td>grid, gr</td>
<td>Avaa vedosarkin: selattava luettelo pienoislautojen ruudukkona, kaksikymmentäneljä kerrallaan sivulla; pienoiskuvan valitseminen avaa sen aseman.</td>
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
<td>rollout, ro [fast|standard]</td>
<td>Rollaa nykyisen aseman (Analyysi-paneelissa valitulla asetuksella tai nimetyllä esiasetuksella) ja avaa Analyysi-paneelin. <code>ro search [fast|standard]</code> käynnistää sen näytettyyn luetteloon vahvistuksen jälkeen, jossa kerrotaan kokonaismäärä; <code>ro stop</code> pysäyttää käynnissä olevan rolloutin.</td>
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
<td>lesson, le [N | edit [N]]</td>
<td>Ilman argumenttia luettelee tietokannan oppitunnit tilarivillä; <code>le N</code> avaa oppitunnin N sen ensimmäisestä vaiheesta; <code>le edit</code> avaa oppituntieditorin, <code>le edit N</code> oppitunnille N (katso Oppitunnit).</td>
</tr>
<tr>
<td>study, sq</td>
<td>Avaa tuonnit ylittävän opiskelujonon: blunderisi, joita mikään ei ole vielä käsitellyt, kalleimmasta halvimpaan (ks. Blunderit, joita mikään ei ole vielä käsitellyt).</td>
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
<td>Lataa pahimmat virheet (equity/MWC) analyysinäkymään nykyisen tilastosuodattimen mukaisesti. Valinnainen luku valitsee, kuinka monta ladataan (<code>bl 50</code>); oletuksena 10.</td>
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
<td>Hae näytetyistä asemista: nykyiset tulokset, avoin kokoelma tai läpikäytävä ottelu.</td>
</tr>
</tbody>
</table>
<h3>Hakusuodattimet</h3>
<p>Tämä taulukko on hakukieliopin viite: komentorivi, suodatinkirjasto ja <code>blunderdb search</code> -komennon valitsin <code>--query</code> lukevat kaikki samoja tunnuksia. Sarake <em>CLI-vastine</em> antaa, silloin kun sellainen on olemassa, saman asian tekevän <code>search</code>-valitsimen (ks. Komentoriviliittymä (CLI)); viiva merkitsee suodatinta, jonka vain kielioppi osaa ilmaista.</p>
<p>Viisi tunnusta ei kanna omaa arvoaan: ne lukevat sen hakulaudalta. <code>cube</code> ja <code>score</code> ottavat sille asetetun kuution ja tuloksen, <code>d</code> päätöksen tyypin, <code>D</code> ja <code>D1</code> nopat, <code>x</code> <em>Paitsi</em>-välilehdellä piirretyn rakenteen. Heittoa ei siis koskaan kirjoiteta tunnukseen: <code>D65</code> ei ole olemassa, ja vain poissulkeva muoto kantaa numeronsa (<code>xD65</code>). Komentorivillä, jossa lautaa ei ole, nämä tunnukset vertautuvat tyhjään lautaan; siellä on käytettävä kolmannen sarakkeen valitsimia.</p>
<p>Yksi ainoa tunnus <strong>järjestää</strong> sen sijaan että rajaisi: <code>like</code> järjestää tuloksen kasvavan etäisyyden mukaan kohdeasemasta, ja kaikki muut tunnukset rajaavat näin järjestettyä joukkoa.</p>
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
<td>dd</td>
<td>Päätös on kuution toiminto tyyppiä Tuplaa / Älä tuplaa (ei Take / Pass -vastaus). Edellyttää kuutiopäätöstä.</td>
<td><code>--cube-response double</code></td>
</tr>
<tr>
<td>dr</td>
<td>Päätös on Take / Pass -vastaus. Edellyttää kuutiopäätöstä; <code>dd</code>:n kanssa <code>dr</code> on etusijalla.</td>
<td><code>--cube-response takepass</code></td>
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
<td>Paikka on kohdattu tietokannassa vähintään x kertaa — siinä tehtyjen päätösten määrä, kaikki ottelut ja kaikki pelaajat yhteensä; <code>pl</code>-, <code>pl!</code>- tai <code>op</code>-suodattimen kanssa lasketaan vain kyseisen pelaajan esiintymät. Muodot <code>n&gt;3</code>, <code>n&lt;2</code>, <code>n3,10</code> ja <code>n4</code> (tasan neljä).</td>
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
<td><code>au'Alice'</code></td>
<td>Positiolla on tämän tekijän allekirjoittama kommentti (koko nimi, isoja ja pieniä kirjaimia erottamatta).</td>
<td><code>--comment-author</code></td>
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
<td>Päivämäärä, jolloin asema lisättiin tietokantaan, x:n (VVVV/KK/PP) jälkeen. Se ei ole ottelun päivämäärä (<code>md</code>): lisääminen määrittää sen ja kahden tietokannan yhdistäminen säilyttää sen.</td>
<td>—</td>
</tr>
<tr>
<td>T&lt;x</td>
<td>Päivämäärä, jolloin asema lisättiin tietokantaan, ennen x:ää (VVVV/KK/PP). Se ei ole ottelun päivämäärä (<code>md</code>).</td>
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
<td>tn'nimi'</td>
<td>Hae turnauksista, joiden nimi on <code>nimi</code>: kirjainkoolla ei ole väliä ja <code>*</code> korvaa minkä tahansa merkkijonon (esim. <code>tn'open*'</code>).</td>
<td><code>--tournament-name</code></td>
</tr>
<tr>
<td>rd:x</td>
<td>Hae kierroksen x otteluista (esim. <code>rd:3</code>, <code>rd:Finaali</code>): kierroksen teksti vertaillaan kirjainkoosta välittämättä, <code>*</code> korvaa minkä tahansa merkkijonon. Toistettavissa (<code>rd:1 rd:2</code>): jompikumpi.</td>
<td><code>--round</code></td>
</tr>
<tr>
<td>ml:x</td>
<td>Ottelun pituus on x pistettä. Muodot <code>ml:7</code>, <code>ml:5,9</code>, <code>ml&gt;5</code> ja <code>ml&lt;9</code> (rajat mukaan lukien).</td>
<td><code>--match-lengths</code></td>
</tr>
<tr>
<td>md:x..y</td>
<td><strong>Ottelun päivämäärä</strong>, luettuna aseman <code>match_date</code>-sarakkeesta (vanhimman asemaan johtavan ottelun päivämäärä), ei analyysin luontipäivä (<code>T</code>). Jokainen raja on vuosi, kuukausi tai päivä (<code>2024</code>, <code>2024-06</code>, <code>2024-06-15</code>) ja kattaa koko kestonsa, rajat mukaan lukien: <code>md:2024-01..2024-12</code> ulottuu 1. tammikuuta 31. joulukuuta 2024. Muodot <code>md:2024</code> (koko vuosi), <code>md&gt;2024-06</code> ja <code>md&lt;2024-06</code>.</td>
<td><code>--match-date</code></td>
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
<td>Hae asemia ottelusta, jossa nimetty pelaaja oli mukana kummalla tahansa puolella (esim. <code>pl'Alice'</code>). Kirjainkoolla ei ole väliä ja <code>*</code> korvaa minkä tahansa merkkijonon (<code>pl'Ali*'</code>).</td>
<td><code>--player</code></td>
</tr>
<tr>
<td><code>pl!'nimi'</code></td>
<td>Vain tämän pelaajan tekemät päätökset: vuorossa oleva pelaaja on se, joka ottelussa istuu tämännimisellä puolella (pelaaja 1 tai pelaaja 2). Samat kirjainkoko- ja jokerimerkkisäännöt kuin <code>pl</code>:llä.</td>
<td><code>--player</code> <code>--seat-only</code></td>
</tr>
<tr>
<td><code>op'nimi'</code></td>
<td>Vain ottelut, joissa tämä pelaaja on <code>pl</code>:n pelaajan vastustaja (<code>pl'Alice' op'Bob'</code>: Alice Bobia vastaan, kummalla puolella tahansa; <code>pl!</code>:n kanssa vain Alicen päätökset). Ilman <code>pl</code>:ää pelkkä <code>op</code> tarkoittaa pelaajaa kummalla puolella tahansa, kuten <code>pl</code>. Samat kirjainkoko- ja jokerimerkkisäännöt.</td>
<td><code>--opponent</code></td>
</tr>
<tr>
<td>pr&gt;x</td>
<td>Päätöksen tehneen pelaajan <strong>ottelun PR</strong> on vähintään x, luettuna ottelukohtaisista tilastoista tämän pelaajan puolelta (ei vastustajan PR); ottelu ilman PR:ää suljetaan pois. Muodot <code>pr&gt;8</code>, <code>pr&lt;5</code> ja <code>pr4,9</code>, rajat mukaan lukien.</td>
<td><code>--pr</code></td>
</tr>
<tr>
<td>ad:xg</td>
<td>Asemalle tallennetun analyysin moottori ja syvyys. Moottorit: <code>xg</code>, <code>gnubg</code>, <code>bgblitz</code>, <code>hedgehog</code>, <code>gammonnet</code> (moottorin nimen alku, kirjainkoolla ei ole väliä). Syvyydet: <code>3ply</code> (täsmälleen 3 ply), <code>3ply+</code> (vähintään 3 ply), <code>book</code> (avauskirja), <code>rollout</code> (rollout, mukaan lukien XG Roller ja Roller++). Toistettavissa: moottorit ovat vaihtoehtoja, samoin syvyydet, ja sekä moottorin että syvyyden on täsmättävä (<code>ad:xg ad:gnubg ad:3ply+</code>).</td>
<td><code>--analysis</code></td>
</tr>
<tr>
<td>like, like42, like&lt;12, like42*</td>
<td>Järjestää tuloksen kasvavan etäisyyden mukaan kohdeasemasta sen sijaan että rajaisi sitä: <code>like</code> ottaa nykyisen aseman, <code>like42</code> sen, jonka indeksi on 42, <code>like&lt;12</code> hylkää kaiken yli kahdentoista nappulapipin päässä olevan, <code>like42*</code> laajentaa kohteen luokan kaikkiin päätöstyyppeihin ja molempiin pelimuotoihin, rahapeliin ja otteluun. Ks. Hakupaneeli.</td>
<td>—</td>
</tr>
</tbody>
</table>
<p>Arvot <code>pl</code>, <code>pl!</code>, <code>op</code>, <code>tn</code>, <code>m</code> ja <code>t</code> alkavat lainausmerkillä tai heittomerkillä ja päättyvät jompaankumpaan. Tunniste, jossa on lainausmerkki mutta joka ei muodosta täydellistä arvoa, ohitetaan, ja <code>blunderdb search --query</code> ilmoittaa siitä. Tunniste (tagi) voi sisältää heittomerkin (<code>#l'ouverture</code>), mutta ei koskaan lainausmerkkiä. Puolipiste erottaa tunnus- ja tagiluettelot: se ei esiinny missään arvossa, eikä <code>ma1;2</code> ole tunniste (kirjoita <code>ma1 ma2</code>).</p>
<h3>Litterointi päätteestä</h3>
<p>Komentorivillä ei ole litterointikomentoa: eleet kirjoitetaan <em>Litterointi</em>-välilehdellä. Sovelluksen ulkopuolella ne kulkevat komennon <code>blunderdb call</code> kautta (Litterointi rajapinnan kautta), esimerkiksi <code>blunderdb call transcriptions.apply --db base.db --if-match 3 --json '{"id":1,"gesture":{"Kind":"validate"}}'</code>. <code>--if-match</code> nimeää edellisen kutsun palauttaman revision; jokainen kutsu on oma istuntonsa, ilman <code>sessionId</code>-tunnistetta ja ilman kumoamista kutsusta toiseen.</p>
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
        Yksipuolinen (6 pistettä, 15 nappulaa, EPC:tä varten) ja kaksipuolinen (6 pistettä, 6 nappulaa, kilpajuoksujen kuutiopäätöksiä varten) bearoff-tietokanta lasketaan blunderDB:ssä itsessään
        <strong>GNU Backgammonin</strong> (GNUbg) <em>makebearoff</em>-työkalun porttauksella; tulos on tavu tavulta identtinen gnubgin tuloksen kanssa, jonka SHA-256-tiiviste toimii viitteenä. GNUbg
        on GPL-lisensoitu vapaa ohjelmisto.
    </li>
    <li>Ottelutiedostot luetaan kirjastoilla <em>xgparser</em>, <em>gnubgparser</em>, <em>bgfparser</em> ja <em>ogxmparser</em> (MIT).</li>
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
