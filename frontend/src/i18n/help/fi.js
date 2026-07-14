// Suomenkielinen ohjesisältö (käännetty tiedostosta en.js).
// Jokainen arvo on vastaavan HelpModal-välilehden sisäinen HTML sellaisenaan.
export default {
    manual: `
                    <h3>Johdanto</h3>
                    <p>
                        blunderDB on ohjelmisto backgammon-asematietokantojen luomiseen. Sen tärkein vahvuus on tarjota yksi paikka, johon koota asemat, joita pelaaja on kohdannut (verkossa,
                        turnauksissa), ja mahdollistaa näiden asemien uudelleentutkiminen suodattamalla niitä erilaisilla mielivaltaisesti yhdisteltävillä suodattimilla. blunderDB:tä voi käyttää myös
                        viiteasemien luettelojen luomiseen.
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
                        <li>ottelujen järjestäminen turnauksiin.</li>
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
                        <li>laskevat EPC-arvoja bearoff-asemille (EPC-paneeli),</li>
                        <li>selaavat tallennettuja hakusuodattimia (Suodatinkirjasto-paneeli),</li>
                        <li>selaavat hakuhistoriaa (Hakuhistoria-paneeli),</li>
                        <li>näyttävät toimintojen lokit (Loki-paneeli).</li>
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
                        <strong>Tab</strong>-näppäimen painaminen avaa hakupaneelin ja mahdollistaa aseman muokkaamisen laudalla sen lisäämiseksi tietokantaan tai haettavan asemarakenteen määrittämiseksi.
                        Nappuloiden jakauman, kuution, pistetilanteen ja vuoron voi muokata hiirellä.
                    </p>

                    <h3>Komentorivi</h3>
                    <p>
                        Tilapalkkiin sisältyvä komentorivi mahdollistaa kaikkien blunderDB:n toimintojen suorittamisen: tietokantatoiminnot, asemanavigoinnin, analyysin ja
                        kommenttien näyttämisen, asemien hakemisen suodattimilla... Käyttöliittymään tutustumisen jälkeen on suositeltavaa siirtyä vähitellen käyttämään komentoriviä, joka mahdollistaa tehokkaan ja
                        sujuvan blunderDB:n käytön, erityisesti asemahakutoiminnoissa.
                    </p>
                    <p>
                        Avaa komentorivi painamalla <strong>välilyönti</strong>-näppäintä. Tilapalkkiin ilmestyy kehote. Kirjoita komentosi ja suorita se painamalla <strong>Enter</strong>. Peruuta painamalla
                        <strong>Escape</strong>.
                        Komentohistoria ja tulokset kirjataan <strong>Loki</strong>-paneeliin.
                    </p>
                    <p>
                        blunderDB suorittaa käyttäjän lähettämät kyselyt edellyttäen, että ne ovat kelvollisia, ja muuttaa tarvittaessa välittömästi tietokannan tilaa. Käyttäjältä ei vaadita erillisiä tallennustoimintoja.
                    </p>
                    <p>
                        Tarkentaaksesi hakua aiemmin suodatettujen asemien sisällä käytä <strong>ss</strong>-komentoa, jota seuraavat suodattimet (esim. <strong>ss nc</strong>). Tämä rajaa haun
                        vain parhaillaan näytettäviin asemiin, mikä mahdollistaa tulosten asteittaisen kaventamisen. Hakupaneeli (<strong>Ctrl+F</strong>) tarjoaa myös "Hae nykyisistä tuloksista" -valintaruudun
                        samaa toiminnallisuutta varten.
                    </p>

                    <h3>EPC-laskin</h3>
                    <p>EPC-laskin (Effective Pip Count) laskee bearoff-asemien efektiivisen pip-luvun. Se käyttää GNUbg:n yksipuolista 6-pisteen bearoff-tietokantaa tarkkoihin EPC-arvoihin.</p>
                    <p>
                        Avaa EPC-paneeli painamalla <strong>Ctrl+E</strong>, napsauttamalla alapaneelin EPC-välilehteä tai kirjoittamalla <strong>epc</strong> komentoriville. Lauta alustetaan vakiomuotoisella
                        bearoff-asetelmalla (15 nappulaa).
                    </p>
                    <p>
                        Voit vapaasti lisätä tai poistaa nappuloita kotialueen pisteille hiirellä. EPC-arvot näytetään reaaliajassa erillisessä EPC-paneelissa, ja niistä näkyy kummankin pelaajan osalta:
                    </p>
                    <ul>
                        <li><strong>EPC</strong>: keskimääräinen pip-määrä, joka tarvitaan kaikkien nappuloiden poistamiseen,</li>
                        <li><strong>Pip-luku</strong>: raaka pip-luku,</li>
                        <li><strong>Wastage</strong>: EPC:n ja pip-luvun erotus,</li>
                        <li><strong>Heittojen ka.</strong>: keskimääräinen heittojen määrä nappuloiden poistamiseen,</li>
                        <li><strong>Keskihajonta</strong>: heittojen määrän keskihajonta.</li>
                    </ul>
                    <p>Kun molemmilla pelaajilla on nappuloita kotialueellaan, vertailuosio näyttää EPC:n ja pip-luvun erot.</p>
                    <p>Sulje EPC-paneeli painamalla <strong>Ctrl+E</strong> uudelleen tai vaihtamalla toiselle välilehdelle.</p>

                    <h3>Ottelunavigointi</h3>
                    <p>
                        blunderDB mahdollistaa tuotujen ottelujen siirtojen selaamisen. Avaa Ottelu-paneeli painamalla <strong>Ctrl+Tab</strong> ja kaksoisnapsauta ottelua (tai paina <strong>Enter</strong>)
                        ladataksesi sen asemat.
                    </p>
                    <p>
                        Ottelua selattaessa viimeksi vierailtu asema tallennetaan ja palautetaan automaattisesti. Käytä <strong>vasen</strong>/<strong>oikea</strong>-näppäimiä asemien välillä liikkumiseen ja
                        <strong>PageUp</strong>/<strong>PageDown</strong> pelien välillä hyppäämiseen.
                    </p>
                    <p>
                        Analyysipaneeli (<strong>Ctrl+L</strong>) näyttää kunkin siirron analyysin, jossa pelattu siirto on korostettu. Paina <strong>d</strong> vaihtaaksesi nappula- ja kuutioanalyysin välillä.
                    </p>

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
                        laudalla. Arvioi muistisi näppäimillä <strong>1</strong> (Uudelleen), <strong>2</strong> (Vaikea), <strong>3</strong> (Hyvä) tai <strong>4</strong> (Helppo). Paina <strong>Esc</strong> lopettaaksesi
                        ja palataksesi pakkalistaan.
                    </p>
                    <p>
                        <strong>Pysäytä/jatka:</strong> Voit pysäyttää kertausistunnon milloin tahansa painamalla <strong>Esc</strong>. Painike muuttuu muotoon <em>Jatka</em> ja näyttää edistymisesi. Napsauta sitä
                        jatkaaksesi siitä, mihin jäit.
                    </p>
                    <p>
                        <strong>Pakkojen hallinta:</strong> Käytä toimintopainikkeita pakkojen nimeämiseen uudelleen, synkronointiin, nollaamiseen tai poistamiseen. FSRS-parametrit (muistitavoite, maksimiväli, fuzz) voidaan määrittää pakkakohtaisesti
                        Asetuksissa (ratas-kuvake).
                    </p>

                    <h3>Turnaukset</h3>
                    <p>Turnaukset mahdollistavat ottelujen ryhmittelyn tapahtuman mukaan. Avaa Turnaus-paneeli painamalla <strong>Ctrl+Y</strong> hallitaksesi turnauksia ja liittääksesi otteluita niihin.</p>

                    <h3>Tilastot</h3>
                    <p>
                        Tilasto-paneeli (<strong>Ctrl+D</strong>) näyttää suoritustilastot (PR ja MWC-kustannus), jotka lasketaan kaikista tuoduista asemista. Käytä suodatinpalkkia rajataksesi analyysiä
                        pelaajan, turnauksen, päivämääräalueen, päätöstyypin tai ottelun pituuden mukaan. Napsauta mitä tahansa mittaria porautuaksesi vastaaviin asemiin.
                    </p>
`,
    shortcuts: `
                    <h3>Tietokanta</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Pikanäppäin</th>
                                <th>Kuvaus</th>
                            </tr>
                        </thead>

                        <tbody>
                            <tr>
                                <td>Ctrl + N</td>
                                <td>Uusi tietokanta</td>
                            </tr>

                            <tr>
                                <td>Ctrl + O</td>
                                <td>Avaa tietokanta</td>
                            </tr>

                            <tr>
                                <td>Ctrl + Shift + I</td>
                                <td>Tuo tietokanta</td>
                            </tr>

                            <tr>
                                <td>Ctrl + Shift + S</td>
                                <td>Vie tietokanta</td>
                            </tr>

                            <tr>
                                <td>Ctrl + Q</td>
                                <td>Poistu blunderDB:stä</td>
                            </tr>

                            <tr>
                                <td>Ctrl + M</td>
                                <td>Muokkaa metatietoja</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Asema</h3>

                    <table>
                        <thead>
                            <tr>
                                <th>Pikanäppäin</th>
                                <th>Kuvaus</th>
                            </tr>
                        </thead>

                        <tbody>
                            <tr>
                                <td>Ctrl + I</td>
                                <td>Tuo asema tai ottelu</td>
                            </tr>

                            <tr>
                                <td>Ctrl + C</td>
                                <td>Kopioi asema (kopioi myös lautaleikepöydälle)</td>
                            </tr>

                            <tr>
                                <td>Ctrl + X</td>
                                <td>Kopioi laudan kuva leikepöydälle (PNG)</td>
                            </tr>

                            <tr>
                                <td>Ctrl + X, Ctrl + X</td>
                                <td>Kopioi lauta + analyysi -kuva leikepöydälle (PNG)</td>
                            </tr>

                            <tr>
                                <td>Ctrl + V</td>
                                <td>Liitä asema (hakupaneelissa: liitä laudalle)</td>
                            </tr>

                            <tr>
                                <td>Ctrl + S</td>
                                <td>Tallenna asema</td>
                            </tr>

                            <tr>
                                <td>Ctrl + U</td>
                                <td>Päivitä asema</td>
                            </tr>

                            <tr>
                                <td>Del</td>
                                <td>Poista asema</td>
                            </tr>

                            <tr>
                                <td>Backspace</td>
                                <td>Nollaa lauta, kuutio, pistetilanne ja nopat</td>
                            </tr>

                            <tr>
                                <td>Ctrl + G</td>
                                <td>Näytä aseman metatiedot</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Navigointi</h3>

                    <table>
                        <thead>
                            <tr>
                                <th>Pikanäppäin</th>
                                <th>Kuvaus</th>
                            </tr>
                        </thead>

                        <tbody>
                            <tr>
                                <td>Ctrl + R</td>
                                <td>Lataa kaikki asemat uudelleen</td>
                            </tr>

                            <tr>
                                <td>PageUp, h</td>
                                <td>Ensimmäinen asema / edellinen peli (ottelunavigointi)</td>
                            </tr>

                            <tr>
                                <td>Left, k</td>
                                <td>Edellinen asema</td>
                            </tr>

                            <tr>
                                <td>Right, j</td>
                                <td>Seuraava asema</td>
                            </tr>

                            <tr>
                                <td>PageDown, l</td>
                                <td>Viimeinen asema / seuraava peli (ottelunavigointi)</td>
                            </tr>

                            <tr>
                                <td>r</td>
                                <td>Lataa satunnainen asema</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Näyttö</h3>

                    <table>
                        <thead>
                            <tr>
                                <th>Pikanäppäin</th>
                                <th>Kuvaus</th>
                            </tr>
                        </thead>

                        <tbody>
                            <tr>
                                <td>Ctrl + ArrowLeft</td>
                                <td>Aseta laudan suunta vasemmalle</td>
                            </tr>

                            <tr>
                                <td>Ctrl + ArrowRight</td>
                                <td>Aseta laudan suunta oikealle</td>
                            </tr>

                            <tr>
                                <td>p</td>
                                <td>Näytä/piilota pip-luku</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Toiminnot</h3>

                    <table>
                        <thead>
                            <tr>
                                <th>Pikanäppäin</th>
                                <th>Kuvaus</th>
                            </tr>
                        </thead>

                        <tbody>
                            <tr>
                                <td>Tab</td>
                                <td>Avaa hakupaneeli (asemaeditori)</td>
                            </tr>

                            <tr>
                                <td>Space</td>
                                <td>Avaa komentorivi</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Työkalut</h3>

                    <table>
                        <thead>
                            <tr>
                                <th>Pikanäppäin</th>
                                <th>Kuvaus</th>
                            </tr>
                        </thead>

                        <tbody>
                            <tr>
                                <td>Ctrl + L</td>
                                <td>Näytä analyysi</td>
                            </tr>

                            <tr>
                                <td>Ctrl + P</td>
                                <td>Kirjoita kommentteja</td>
                            </tr>

                            <tr>
                                <td>Ctrl + K</td>
                                <td>Näytä Anki-paneeli</td>
                            </tr>

                            <tr>
                                <td>Ctrl + F</td>
                                <td>Hakupaneeli</td>
                            </tr>

                            <tr>
                                <td>Ctrl + Tab</td>
                                <td>Ottelu-paneeli</td>
                            </tr>

                            <tr>
                                <td>Ctrl + B</td>
                                <td>Kokoelma-paneeli</td>
                            </tr>

                            <tr>
                                <td>Ctrl + Y</td>
                                <td>Turnaukset-paneeli</td>
                            </tr>

                            <tr>
                                <td>Ctrl + D</td>
                                <td>Tilasto-paneeli</td>
                            </tr>

                            <tr>
                                <td>Ctrl + E</td>
                                <td>EPC-paneeli</td>
                            </tr>

                            <tr>
                                <td>?</td>
                                <td>Avaa ohje</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Komentorivi</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Pikanäppäin</th>
                                <th>Kuvaus</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>Up</td>
                                <td>Selaa komentohistoriaa ylöspäin</td>
                            </tr>
                            <tr>
                                <td>Down</td>
                                <td>Selaa komentohistoriaa alaspäin</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Analyysipaneeli</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Pikanäppäin</th>
                                <th>Kuvaus</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>Click</td>
                                <td>Valitse/poista valinta siirrosta (näytä/piilota nuolet)</td>
                            </tr>
                            <tr>
                                <td>Up, k</td>
                                <td>Valitse edellinen siirto (kun siirto on valittuna)</td>
                            </tr>
                            <tr>
                                <td>Down, j</td>
                                <td>Valitse seuraava siirto (kun siirto on valittuna)</td>
                            </tr>
                            <tr>
                                <td>d</td>
                                <td>Vaihda nappula- ja kuutioanalyysin välillä (vain ottelunavigointi)</td>
                            </tr>
                            <tr>
                                <td>Esc</td>
                                <td>Poista siirron valinta. Jos siirtoa ei ole valittu, sulje paneeli.</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Hakuhistoria-paneeli</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Pikanäppäin</th>
                                <th>Kuvaus</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>Click</td>
                                <td>Valitse/poista valinta hausta (näytä asema)</td>
                            </tr>
                            <tr>
                                <td>Double-click</td>
                                <td>Suorita haku ja sulje paneeli</td>
                            </tr>
                            <tr>
                                <td>Up, k</td>
                                <td>Valitse edellinen haku (uudempi, ylempänä)</td>
                            </tr>
                            <tr>
                                <td>Down, j</td>
                                <td>Valitse seuraava haku (vanhempi, alempana)</td>
                            </tr>
                            <tr>
                                <td>Esc</td>
                                <td>Poista haun valinta. Jos hakua ei ole valittu, sulje paneeli.</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Suodatinkirjasto-paneeli</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Pikanäppäin</th>
                                <th>Kuvaus</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>Click</td>
                                <td>Valitse/poista valinta suodattimesta (näytä asema)</td>
                            </tr>
                            <tr>
                                <td>Double-click</td>
                                <td>Suorita suodatinhaku</td>
                            </tr>
                            <tr>
                                <td>Up, k</td>
                                <td>Valitse edellinen suodatin (kun suodatin on valittuna)</td>
                            </tr>
                            <tr>
                                <td>Down, j</td>
                                <td>Valitse seuraava suodatin (kun suodatin on valittuna)</td>
                            </tr>
                            <tr>
                                <td>Esc</td>
                                <td>Poista suodattimen valinta. Jos suodatinta ei ole valittu, sulje paneeli.</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Ottelu-paneeli</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Pikanäppäin</th>
                                <th>Kuvaus</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>Click</td>
                                <td>Valitse/poista valinta ottelusta</td>
                            </tr>
                            <tr>
                                <td>Double-click</td>
                                <td>Lataa ottelun asemat</td>
                            </tr>
                            <tr>
                                <td>Up, k</td>
                                <td>Valitse edellinen ottelu</td>
                            </tr>
                            <tr>
                                <td>Down, j</td>
                                <td>Valitse seuraava ottelu</td>
                            </tr>
                            <tr>
                                <td>Del</td>
                                <td>Poista valittu ottelu</td>
                            </tr>
                            <tr>
                                <td>Esc</td>
                                <td>Poista ottelun valinta. Jos ottelua ei ole valittu, sulje paneeli.</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Kokoelma-paneeli</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Pikanäppäin</th>
                                <th>Kuvaus</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>Click</td>
                                <td>Valitse kokoelma (näytä sen asemat)</td>
                            </tr>
                            <tr>
                                <td>Double-click</td>
                                <td>Avaa kokoelma ja selaa sen asemia</td>
                            </tr>
                            <tr>
                                <td>Del</td>
                                <td>Poista nykyinen asema aktiivisesta kokoelmasta</td>
                            </tr>
                            <tr>
                                <td>Esc</td>
                                <td>Poista kokoelman valinta. Jos kokoelmaa ei ole valittu, sulje paneeli.</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Anki-paneeli</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Pikanäppäin</th>
                                <th>Kuvaus</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>1</td>
                                <td>Arvio: Uudelleen (kertaus epäonnistui, näytä pian uudelleen)</td>
                            </tr>
                            <tr>
                                <td>2</td>
                                <td>Arvio: Vaikea (vaikea muistaa)</td>
                            </tr>
                            <tr>
                                <td>3</td>
                                <td>Arvio: Hyvä (oikein muistettu)</td>
                            </tr>
                            <tr>
                                <td>4</td>
                                <td>Arvio: Helppo (vaivaton muistaa)</td>
                            </tr>
                            <tr>
                                <td>Esc</td>
                                <td>Lopeta kertaus ja palaa pakkalistaan (voi jatkaa myöhemmin)</td>
                            </tr>
                        </tbody>
                    </table>
`,
    commands: `
                    <h3>Tietokanta</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Komento</th>
                                <th>Kuvaus</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>new, ne, n</td>
                                <td>Luo uusi tietokanta</td>
                            </tr>
                            <tr>
                                <td>open, op, o</td>
                                <td>Avaa olemassa oleva tietokanta</td>
                            </tr>
                            <tr>
                                <td>import_db, idb</td>
                                <td>Tuo ja yhdistä toinen tietokanta</td>
                            </tr>
                            <tr>
                                <td>export_db, edb</td>
                                <td>Vie nykyinen valinta uuteen tietokantaan</td>
                            </tr>
                            <tr>
                                <td>quit, q</td>
                                <td>Poistu blunderDB:stä</td>
                            </tr>
                            <tr>
                                <td>meta</td>
                                <td>Muokkaa metatietoja</td>
                            </tr>
                        </tbody>
                    </table>
                    <h3>Asema</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Komento</th>
                                <th>Kuvaus</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>import, i</td>
                                <td>Tuo asema tai ottelu</td>
                            </tr>
                            <tr>
                                <td>write, wr, w</td>
                                <td>Tallenna asema</td>
                            </tr>
                            <tr>
                                <td>write!, wr!, w!</td>
                                <td>Päivitä asema</td>
                            </tr>
                            <tr>
                                <td>delete, del, d</td>
                                <td>Poista asema</td>
                            </tr>
                        </tbody>
                    </table>
                    <h3>Navigointi</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Komento</th>
                                <th>Kuvaus</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>[numero]</td>
                                <td>Siirry tiettyyn asemaan indeksin mukaan</td>
                            </tr>
                        </tbody>
                    </table>
                    <h3>Työkalut</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Komento</th>
                                <th>Kuvaus</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>list, l</td>
                                <td>Näytä analyysi</td>
                            </tr>
                            <tr>
                                <td>comment, co</td>
                                <td>Kirjoita kommentteja</td>
                            </tr>
                            <tr>
                                <td>filter, fl</td>
                                <td>Näytä suodatinkirjasto</td>
                            </tr>
                            <tr>
                                <td>history, hi</td>
                                <td>Näytä hakuhistoria</td>
                            </tr>
                            <tr>
                                <td>match, ma</td>
                                <td>Näytä Ottelu-paneeli</td>
                            </tr>
                            <tr>
                                <td>collection, coll</td>
                                <td>Näytä Kokoelmat-paneeli</td>
                            </tr>
                            <tr>
                                <td>epc</td>
                                <td>EPC-laskin (Effective Pip Count)</td>
                            </tr>
                            <tr>
                                <td>m</td>
                                <td>Navigoi viimeksi vierailtu ottelu</td>
                            </tr>
                            <tr>
                                <td>help, he, h</td>
                                <td>Avaa ohje</td>
                            </tr>
                            <tr>
                                <td>met</td>
                                <td>Avaa Kazaross-XG2-otteluekvivalenssitaulukko</td>
                            </tr>
                            <tr>
                                <td>tp2</td>
                                <td>2-kuution take point (Live ja Last)</td>
                            </tr>
                            <tr>
                                <td>tp2_live</td>
                                <td>2-kuution take point pitkissä juoksuissa</td>
                            </tr>
                            <tr>
                                <td>tp2_last</td>
                                <td>2-kuution take point viimeisen heiton asemissa</td>
                            </tr>
                            <tr>
                                <td>tp4</td>
                                <td>4-kuution take point (Live ja Last)</td>
                            </tr>
                            <tr>
                                <td>tp4_live</td>
                                <td>4-kuution take point pitkissä juoksuissa</td>
                            </tr>
                            <tr>
                                <td>tp4_last</td>
                                <td>4-kuution take point viimeisen heiton asemissa</td>
                            </tr>
                            <tr>
                                <td>gv1</td>
                                <td>1-kuution gammon-arvot</td>
                            </tr>
                            <tr>
                                <td>gv2</td>
                                <td>2-kuution gammon-arvot</td>
                            </tr>
                            <tr>
                                <td>gv4</td>
                                <td>4-kuution gammon-arvot</td>
                            </tr>
                            <tr>
                                <td>#tag1 tag2 ...</td>
                                <td>Merkitse asema tunnisteilla</td>
                            </tr>
                            <tr>
                                <td>s</td>
                                <td>Hae asemia suodattimilla</td>
                            </tr>
                            <tr>
                                <td>ss</td>
                                <td>Hae nykyisistä tuloksista suodattimilla</td>
                            </tr>
                            <tr>
                                <td>e</td>
                                <td>Lataa kaikki asemat uudelleen</td>
                            </tr>
                        </tbody>
                    </table>
                    <h3>Suodattimet</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Suodatin</th>
                                <th>Kuvaus</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>cube, cub, cu, c</td>
                                <td>Sisällytä kuutio</td>
                            </tr>
                            <tr>
                                <td>score, sco, sc, s</td>
                                <td>Sisällytä pistetilanne</td>
                            </tr>
                            <tr>
                                <td>d</td>
                                <td>Sisällytä päätöstyyppi</td>
                            </tr>
                            <tr>
                                <td>D</td>
                                <td>Sisällytä noppaheitto</td>
                            </tr>
                            <tr>
                                <td>nc</td>
                                <td>Sisällytä kontaktittomat asemat</td>
                            </tr>
                            <tr>
                                <td>M</td>
                                <td>Sisällytä peilatut asemat</td>
                            </tr>
                            <tr>
                                <td>i</td>
                                <td>Vain erikseen tuodut asemat, ei ottelun tuonnin mukanaan tuomat</td>
                            </tr>
                            <tr>
                                <td>x</td>
                                <td
                                    >Sulje pois asemat, joissa on <em>mikä tahansa</em> nappula "Except"-välilehdellä piirretystä rakenteesta (esim. nappuloiden piirtäminen pisteille 1, 3 ja 5 pitää vain asemat, joissa ei ole yhtäkään
                                    niistä). Vaihda "At least" / "Except" suodattimien yläpuolelta piirtääksesi poissuljettavat nappulat laudalle (näytetään punaisella merkillä). Pistekohtaista määrää ei rajoiteta (3 pisteellä
                                    sulkee pois 3+ siellä — tehty piste ilman varanappulaa), ja kaksi nopeaa napsautusta pisteellä merkitsee sen pakollisesti tyhjäksi (punainen viivoitettu solu, mikä tahansa väri); yksittäinen napsautus tällä
                                    pisteellä vapauttaa sen. Jaetulla pisteellä "Except" ohittaa "At least":n, kun nämä kaksi ovat ristiriidassa.</td
                                >
                            </tr>
                            <tr>
                                <td>p>x</td>
                                <td>Pip-luku &gt; x</td>
                            </tr>
                            <tr>
                                <td>p&lt;x</td>
                                <td>Pip-luku &lt; x</td>
                            </tr>
                            <tr>
                                <td>px,y</td>
                                <td>Pip-luku välillä x ja y</td>
                            </tr>
                            <tr>
                                <td>P>x</td>
                                <td>Pelaajan absoluuttinen pip-luku &gt; x</td>
                            </tr>
                            <tr>
                                <td>P&lt;x</td>
                                <td>Pelaajan absoluuttinen pip-luku &lt; x</td>
                            </tr>
                            <tr>
                                <td>Px,y</td>
                                <td>Pelaajan absoluuttinen pip-luku välillä x ja y</td>
                            </tr>
                            <tr>
                                <td>e>x</td>
                                <td>Equity &gt; x (millipisteissä)</td>
                            </tr>
                            <tr>
                                <td>e&lt;x</td>
                                <td>Equity &lt; x (millipisteissä)</td>
                            </tr>
                            <tr>
                                <td>ex,y</td>
                                <td>Equity välillä x ja y (millipisteissä)</td>
                            </tr>
                            <tr>
                                <td>E>x</td>
                                <td>Pelaajan 1 siirtovirhe &gt; x (millipisteissä)</td>
                            </tr>
                            <tr>
                                <td>E&lt;x</td>
                                <td>Pelaajan 1 siirtovirhe &lt; x (millipisteissä)</td>
                            </tr>
                            <tr>
                                <td>Ex,y</td>
                                <td>Pelaajan 1 siirtovirhe välillä x ja y (millipisteissä)</td>
                            </tr>
                            <tr>
                                <td>w>x</td>
                                <td>Voittoprosentti &gt; x</td>
                            </tr>
                            <tr>
                                <td>w&lt;x</td>
                                <td>Voittoprosentti &lt; x</td>
                            </tr>
                            <tr>
                                <td>wx,y</td>
                                <td>Voittoprosentti välillä x ja y</td>
                            </tr>
                            <tr>
                                <td>g>x</td>
                                <td>Gammon-prosentti &gt; x</td>
                            </tr>
                            <tr>
                                <td>g&lt;x</td>
                                <td>Gammon-prosentti &lt; x</td>
                            </tr>
                            <tr>
                                <td>gx,y</td>
                                <td>Gammon-prosentti välillä x ja y</td>
                            </tr>
                            <tr>
                                <td>b>x</td>
                                <td>Backgammon-prosentti &gt; x</td>
                            </tr>
                            <tr>
                                <td>b&lt;x</td>
                                <td>Backgammon-prosentti &lt; x</td>
                            </tr>
                            <tr>
                                <td>bx,y</td>
                                <td>Backgammon-prosentti välillä x ja y</td>
                            </tr>
                            <tr>
                                <td>W>x</td>
                                <td>Vastustajan voittoprosentti &gt; x</td>
                            </tr>
                            <tr>
                                <td>W&lt;x</td>
                                <td>Vastustajan voittoprosentti &lt; x</td>
                            </tr>
                            <tr>
                                <td>Wx,y</td>
                                <td>Vastustajan voittoprosentti välillä x ja y</td>
                            </tr>
                            <tr>
                                <td>G>x</td>
                                <td>Vastustajan gammon-prosentti &gt; x</td>
                            </tr>
                            <tr>
                                <td>G&lt;x</td>
                                <td>Vastustajan gammon-prosentti &lt; x</td>
                            </tr>
                            <tr>
                                <td>Gx,y</td>
                                <td>Vastustajan gammon-prosentti välillä x ja y</td>
                            </tr>
                            <tr>
                                <td>B>x</td>
                                <td>Vastustajan backgammon-prosentti &gt; x</td>
                            </tr>
                            <tr>
                                <td>B&lt;x</td>
                                <td>Vastustajan backgammon-prosentti &lt; x</td>
                            </tr>
                            <tr>
                                <td>Bx,y</td>
                                <td>Vastustajan backgammon-prosentti välillä x ja y</td>
                            </tr>
                            <tr>
                                <td>o>x</td>
                                <td>Pelaajan poistetut nappulat &gt; x</td>
                            </tr>
                            <tr>
                                <td>o&lt;x</td>
                                <td>Pelaajan poistetut nappulat &lt; x</td>
                            </tr>
                            <tr>
                                <td>ox,y</td>
                                <td>Pelaajan poistetut nappulat välillä x ja y</td>
                            </tr>
                            <tr>
                                <td>O>x</td>
                                <td>Vastustajan poistetut nappulat &gt; x</td>
                            </tr>
                            <tr>
                                <td>O&lt;x</td>
                                <td>Vastustajan poistetut nappulat &lt; x</td>
                            </tr>
                            <tr>
                                <td>Ox,y</td>
                                <td>Vastustajan poistetut nappulat välillä x ja y</td>
                            </tr>
                            <tr>
                                <td>k>x</td>
                                <td>Pelaajan takanappulat &gt; x</td>
                            </tr>
                            <tr>
                                <td>k&lt;x</td>
                                <td>Pelaajan takanappulat &lt; x</td>
                            </tr>
                            <tr>
                                <td>kx,y</td>
                                <td>Pelaajan takanappulat välillä x ja y</td>
                            </tr>
                            <tr>
                                <td>K>x</td>
                                <td>Vastustajan takanappulat &gt; x</td>
                            </tr>
                            <tr>
                                <td>K&lt;x</td>
                                <td>Vastustajan takanappulat &lt; x</td>
                            </tr>
                            <tr>
                                <td>Kx,y</td>
                                <td>Vastustajan takanappulat välillä x ja y</td>
                            </tr>
                            <tr>
                                <td>z>x</td>
                                <td>Pelaajan nappulat vyöhykkeellä &gt; x</td>
                            </tr>
                            <tr>
                                <td>z&lt;x</td>
                                <td>Pelaajan nappulat vyöhykkeellä &lt; x</td>
                            </tr>
                            <tr>
                                <td>zx,y</td>
                                <td>Pelaajan nappulat vyöhykkeellä välillä x ja y</td>
                            </tr>
                            <tr>
                                <td>Z>x</td>
                                <td>Vastustajan nappulat vyöhykkeellä &gt; x</td>
                            </tr>
                            <tr>
                                <td>Z&lt;x</td>
                                <td>Vastustajan nappulat vyöhykkeellä &lt; x</td>
                            </tr>
                            <tr>
                                <td>Zx,y</td>
                                <td>Vastustajan nappulat vyöhykkeellä välillä x ja y</td>
                            </tr>
                            <tr>
                                <td>bo>x</td>
                                <td>Pelaajan ulkokentän blotti &gt; x</td>
                            </tr>
                            <tr>
                                <td>bo&lt;x</td>
                                <td>Pelaajan ulkokentän blotti &lt; x</td>
                            </tr>
                            <tr>
                                <td>box,y</td>
                                <td>Pelaajan ulkokentän blotti välillä x ja y</td>
                            </tr>
                            <tr>
                                <td>BO>x</td>
                                <td>Vastustajan ulkokentän blotti &gt; x</td>
                            </tr>
                            <tr>
                                <td>BO&lt;x</td>
                                <td>Vastustajan ulkokentän blotti &lt; x</td>
                            </tr>
                            <tr>
                                <td>BOx,y</td>
                                <td>Vastustajan ulkokentän blotti välillä x ja y</td>
                            </tr>
                            <tr>
                                <td>bj&gt;x</td>
                                <td>Pelaajan Jan-blotti &gt; x</td>
                            </tr>
                            <tr>
                                <td>bj&lt;x</td>
                                <td>Pelaajan Jan-blotti &lt; x</td>
                            </tr>
                            <tr>
                                <td>bjx,y</td>
                                <td>Pelaajan Jan-blotti välillä x ja y</td>
                            </tr>
                            <tr>
                                <td>BJ&gt;x</td>
                                <td>Vastustajan Jan-blotti &gt; x</td>
                            </tr>
                            <tr>
                                <td>BJ&lt;x</td>
                                <td>Vastustajan Jan-blotti &lt; x</td>
                            </tr>
                            <tr>
                                <td>BJx,y</td>
                                <td>Vastustajan Jan-blotti välillä x ja y</td>
                            </tr>

                            <tr>
                                <td>t"sana1;sana2;..."</td>
                                <td>Tekstihaku</td>
                            </tr>
                            <tr>
                                <td>m"kuvio1;kuvio2;..."</td>
                                <td>Parhaat siirrot, jotka sisältävät vähintään yhden annetuista kuvioista</td>
                            </tr>
                            <tr>
                                <td>m"ND;DT;DP;..."</td>
                                <td>Paras kuutiopäätös: No Double/Take, Double/Take, Double/Pass</td>
                            </tr>
                            <tr>
                                <td>T&gt;x</td>
                                <td>Luontipäivä &gt; x (vuosi/kuukausi/päivä)</td>
                            </tr>
                            <tr>
                                <td>T&lt;x</td>
                                <td>Luontipäivä &lt; x (vuosi/kuukausi/päivä)</td>
                            </tr>
                            <tr>
                                <td>Tx,y</td>
                                <td>Luontipäivä välillä x ja y</td>
                            </tr>
                            <tr>
                                <td>max</td>
                                <td>Hae ottelusta, jonka ID on x (esim. ma3)</td>
                            </tr>
                            <tr>
                                <td>max,y</td>
                                <td>Hae otteluista, joiden ID:t ovat x:stä y:hyn (esim. ma2,5)</td>
                            </tr>
                            <tr>
                                <td>tnx</td>
                                <td>Hae turnauksesta, jonka ID on x (esim. tn1)</td>
                            </tr>
                            <tr>
                                <td>tnx,y</td>
                                <td>Hae turnauksista, joiden ID:t ovat x:stä y:hyn (esim. tn1,3)</td>
                            </tr>
                        </tbody>
                    </table>
                    <h3>Sekalaista</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Komento</th>
                                <th>Kuvaus</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>clear, cl</td>
                                <td>Tyhjennä komentohistoria</td>
                            </tr>
                            <tr>
                                <td>migrate_from_1_0_to_1_1</td>
                                <td>Migratoi tietokanta versiosta 1.0 versioon 1.1</td>
                            </tr>
                            <tr>
                                <td>migrate_from_1_1_to_1_2</td>
                                <td>Migratoi tietokanta versiosta 1.1 versioon 1.2</td>
                            </tr>
                            <tr>
                                <td>migrate_from_1_2_to_1_3</td>
                                <td>Migratoi tietokanta versiosta 1.2 versioon 1.3</td>
                            </tr>
                        </tbody>
                    </table>
`,
    about: `
                    <h3>Versio</h3>
                    <p>Sovelluksen versio: {appVersion}</p>
                    <p>Tietokannan versio: {dbVersion}</p>

                    <h3>Tekijä</h3>
                    <p><strong>Kévin Unger &lt;blunderdb@proton.me&gt;</strong></p>
                    <p>Minut löytää myös Heroesista nimimerkillä <strong>postmanpat</strong>.</p>
                    <p>
                        Kehitin blunderDB:n alun perin omaan käyttööni havaitakseni kaavoja virheissäni. Mutta on erittäin mukavaa saada palautetta, etenkin kun suunnitteluun, koodaamiseen ja virheenkorjaukseen on käytetty
                        paljon tunteja... Joten kirjoita minulle vapaasti jakaaksesi palautteesi.
                    </p>
                    <p>Tässä useita tapoja ottaa yhteyttä:</p>
                    <ul>
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
                            <strong>Tristan Remille</strong>, joka esitteli minulle backgammonin ilolla ja ystävällisyydellä; joka näytti Tien tämän upean pelin ymmärtämiseen; joka jatkaa
                            tukemistani huolimatta huonoista yrityksistäni pelata paremmin.
                        </li>
                        <li><strong>Nicolas Harmand</strong>, iloinen seuralainen yli vuosikymmenen ajan suurissa seikkailuissa ja loistava pelikumppani siitä lähtien, kun hän sai backgammon-kärpäsen.</li>
                    </ul>
                    <p>Kazaross-XG2-otteluekvivalenssitaulukon (MET) ansio kuuluu <strong>Neil Kazarossille</strong>.</p>
                    <p>Take point- ja gammon-arvotaulukot on otettu <strong>Dirk Schiemannin</strong> kirjasta <em>The Theory of Backgammon</em>.</p>
                    <p>
                        EPC-laskennassa (Effective Pip Count) käytetty yksipuolinen 6-pisteen bearoff-tietokanta on luotu <strong>GNU Backgammonilla</strong> (GNUbg). GNUbg on ilmainen ja avoimen lähdekoodin
                        backgammon-ohjelma, joka on lisensoitu GNU General Public License -lisenssillä.
                    </p>
`
};
