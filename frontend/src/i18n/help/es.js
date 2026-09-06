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
<h3>Introducción</h3>
<p>blunderDB es un programa para crear bases de datos de posiciones de backgammon. Su principal fortaleza es ofrecer un único lugar donde agregar las posiciones que un jugador ha encontrado (en línea, en torneos) y poder reestudiarlas filtrándolas según diversos filtros combinables de forma arbitraria. blunderDB también puede usarse para crear catálogos de posiciones de referencia.</p>
<p>Las posiciones se almacenan en una base de datos representada por un archivo <em>.db</em>. La aplicación de escritorio abre este archivo directamente, nunca una dirección de red: el modo servidor (Modo headless (servidor)) es otro modo del mismo binario, y se pasa de uno a otro exportando o migrando la base, no apuntando la aplicación hacia una URL.</p>
<h3>Interacciones principales</h3>
<p>Las principales interacciones posibles con blunderDB son:</p>
<ul>
<li>añadir una nueva posición,</li>
<li>modificar una posición existente,</li>
<li>copiar la imagen del tablero al portapapeles (PNG) mediante <strong>CTRL-X</strong>, o con el análisis completo mediante <strong>CTRL-X CTRL-X</strong>,</li>
<li>eliminar una posición existente,</li>
<li>buscar una o varias posiciones,</li>
<li>importar partidas desde diferentes fuentes (XG, GNUbg, BGBlitz, Jellyfish), incluidos los comentarios de los ficheros XG,</li>
<li>navegar por las jugadas de una partida importada,</li>
<li>organizar las posiciones en colecciones,</li>
<li>organizar las partidas en torneos.</li>
</ul>
<p>El usuario puede etiquetar libremente las posiciones mediante etiquetas y anotarlas con comentarios.</p>
<h3>Descripción de la interfaz</h3>
<p>La interfaz de blunderDB se compone, de arriba abajo, de:</p>
<ul>
<li>[arriba] la barra de herramientas, que reúne todas las principales operaciones que pueden realizarse sobre la base de datos,</li>
<li>[en el centro] la zona de visualización principal, que permite mostrar o editar posiciones de backgammon,</li>
<li>[abajo] la barra de estado, que presenta diversa información sobre la base de datos o la posición actual, e integra la línea de comandos.</li>
</ul>
<p>Pueden mostrarse paneles para:</p>
<ul>
<li>mostrar los datos de análisis asociados a la posición actual procedentes de eXtreme Gammon (XG), GNUbg o BGBlitz,</li>
<li>mostrar, añadir o modificar comentarios,</li>
<li>buscar y filtrar posiciones según criterios combinables,</li>
<li>mostrar y gestionar las colecciones de posiciones (panel de colecciones),</li>
<li>mostrar la lista de partidas importadas y navegar por las jugadas de una partida (panel de partidas),</li>
<li>mostrar y gestionar los torneos (panel de torneos),</li>
<li>mostrar las estadísticas de rendimiento (panel Stats),</li>
<li>calcular el EPC (Effective Pip Count) de una posición de bearoff (panel Eval),</li>
<li>estudiar las posiciones mediante repetición espaciada (panel Anki),</li>
<li>ver los metadatos de la base de datos (panel Metadatos).</li>
</ul>
<p>Pueden mostrarse ventanas modales para:</p>
<ul>
<li>mostrar la ayuda de blunderDB,</li>
<li>mostrar el catálogo de visitas guiadas (ver Visitas guiadas y base de ejemplo),</li>
<li>configurar la exportación de la base de datos,</li>
<li>configurar blunderDB, en particular el idioma de la interfaz (véase Configuración).</li>
</ul>
<p>La zona de visualización principal pone a disposición del usuario:</p>
<ul>
<li>un tablero para mostrar o editar una posición de backgammon,</li>
<li>el nivel y el propietario del cubo,</li>
<li>el pip count de cada jugador,</li>
<li>la puntuación de cada jugador,</li>
<li>los dados a jugar. Si no se muestra ningún valor en los dados, la posición de los dados indica qué jugador tiene el turno y que la posición es una decisión de cubo. Cuando la decisión de cubo es una respuesta a un doblaje (aceptar/pasar), el cubo propuesto se muestra en el centro del tablero, con el valor ofrecido.</li>
</ul>
<p>Un clic derecho en el tablero abre un menú contextual que ofrece: evaluar la posición mostrada en el panel Eval, evaluar su espejo, copiar la imagen del tablero con su análisis al portapapeles (el equivalente de <em>CTRL-X CTRL-X</em>, menos fácil de descubrir), <strong>guardar la imagen en un fichero</strong> en SVG o PNG, abrir una nueva vista sobre esta posición, y — si la posición ya viene de la base — añadirla a un mazo Anki (repetición espaciada).</p>
<p>El portapapeles es el gesto corriente; guardar es la otra necesidad — la ilustración de un artículo, de un mensaje de foro, de una lección. El <strong>SVG</strong> se ofrece porque el tablero lo es: es la forma que sobrevive a una ampliación, la que se pone en un documento sin desenfocarla. El PNG deriva de él, igual que la copia al portapapeles: un solo renderizado, tres destinos, así que ninguno puede divergir de los demás. Este menú no aparece en el panel Eval ni en el panel Búsqueda, donde el botón derecho ya sirve para poner las fichas del otro color. Véase Llevar una posición al panel Eval para llevar una posición al panel Eval.</p>
<p>La barra de estado está estructurada de izquierda a derecha con la siguiente información:</p>
<ul>
<li>la línea de comandos, accesible pulsando la tecla <em>ESPACIO</em>,</li>
<li>un mensaje informativo relacionado con una operación realizada por el usuario,</li>
<li>el índice de la posición actual, seguido del número de posiciones en la biblioteca actual (o la información de jugada/partida al navegar por un partido),</li>
<li>el <strong>contador de biblioteca</strong> — «412 posiciones · 38 blunders · 5 partidos» — donde cada número <strong>abre lo que cuenta</strong>: las posiciones, la búsqueda <code>E&gt;100</code> preparada en la línea de comandos, o la lista de partidos. Una cifra que no se puede seguir es una decoración. El umbral de blunders es el de las estadísticas, cien milipuntos: dos umbrales harían que la misma palabra dijera dos cosas.</li>
</ul>
<div class="admonition note">
<p>En el caso de posiciones resultantes de una búsqueda del usuario, el número de posiciones indicado en la barra de estado corresponde al número de posiciones filtradas.</p>
</div>
<p>La pestaña <strong>Anki</strong> lleva una <strong>insignia</strong> cuando hay tarjetas por repasar, en todos los mazos. Esa cifra es la razón de abrir la pestaña; no tiene nada que hacer detrás de ella. Cero no muestra nada: una insignia que dice «0» es ruido.</p>
<p>El comando <code>log</code> abre el <strong>registro de actividad</strong>: las últimas doscientas líneas del archivo de registro, un botón para copiarlas — lo necesario para adjuntar un informe a un aviso — y otro para abrir la carpeta que las contiene. El registro no se filtra ni se reformatea: un registro que se embellece es un registro que ya no se puede citar.</p>
<p>En el <strong>historial de búsquedas</strong> del panel Búsqueda, cada token de un comando guardado se muestra como una etiqueta con nombre — <em>Sin contacto</em>, <em>Error de jugada</em> — en vez de un token pelado. El comando exacto queda en la ayuda emergente, porque es el que se relanza; y un token que blunderDB no reconoce se muestra <strong>tal cual</strong> en vez de traducido a lo más parecido.</p>
<h3>Pestañas de vistas</h3>
<p>Bajo la barra de herramientas, una barra de pestañas permite trabajar con varias <strong>vistas</strong> en paralelo. Cada vista es un espacio de trabajo independiente que conserva su propia lista de posiciones, el índice de la posición actual, la posición mostrada, el análisis y la jugada seleccionada, el panel activo, el comentario en curso, así como el contexto de navegación en una partida. Así es posible, por ejemplo, mantener una búsqueda abierta en una vista mientras se recorre una partida en otra.</p>
<ul>
<li><strong>Crear una vista</strong>: hacer clic en el botón <em>+</em> de la barra de pestañas o pulsar <em>CTRL-T</em>. La nueva vista comienza como una copia de la vista actual.</li>
<li><strong>Cerrar una vista</strong>: hacer clic en la cruz de la pestaña o pulsar <em>CTRL-W</em>. La última vista no se puede cerrar.</li>
<li><strong>Cambiar de vista</strong>: hacer clic en una pestaña, pulsar <em>CTRL-PageUp</em> / <em>CTRL-PageDown</em> (o <em>MAJ-J</em> / <em>MAJ-K</em>) para pasar a la vista anterior / siguiente, o <em>CTRL-1</em> a <em>CTRL-9</em> para ir directamente a la n-ésima vista.</li>
<li><strong>Renombrar una vista</strong>: hacer doble clic en la pestaña, escribir el nuevo nombre y validar con <em>INTRO</em>.</li>
</ul>
<p>Las vistas se guardan con el estado de sesión de la base de datos y se restauran al reabrirla.</p>
<h3>Configuración</h3>
<p>El botón de configuración (icono de engranaje) situado en la barra de herramientas, a la izquierda del botón de ayuda, abre la ventana de configuración de blunderDB. Está organizada en seis pestañas:</p>
<ul>
<li><strong>Interfaz</strong> — idioma, escala de visualización, posición del panel;</li>
<li><strong>Colores del tablero</strong> — los colores del tablero;</li>
<li><strong>Bearoff</strong> — las tablas de bearoff utilizadas por el panel Eval;</li>
<li><strong>gammonNet</strong> — los ajustes del evaluador integrado, descritos más abajo;</li>
<li><strong>Carpeta vigilada</strong> — la importación automática de los partidos que llegan a una carpeta, descrita más abajo;</li>
<li><strong>Identidad del emisor</strong> — la clave que firma tus marcas de origen, descrita en la sección Distribuir una base de datos: origen y contraseña.</li>
</ul>
<p>La pestaña <em>Interfaz</em> empieza con un <strong>tema</strong>: <em>seguir el sistema</em>, <em>claro</em>, <em>oscuro</em>, <em>contraste alto</em> o <em>imprimible</em>. El tema ajusta los colores de la interfaz y <strong>propone una paleta de tablero</strong> — una interfaz oscura alrededor de un tablero claro no es un tema oscuro, es la mitad de uno, ya que el tablero ocupa la mayor parte de la ventana.</p>
<p>Usted conserva la última palabra, y el mecanismo lo garantiza en vez de prometerlo: la pestaña <em>Colores</em> sigue ajustando el tablero directamente, y un color elegido después del tema es suyo. Al arrancar solo se aplican los tokens de la interfaz, nunca la paleta del tablero — la que usted ha ajustado ya está cargada, y reescribirla en cada lanzamiento borraría su trabajo una sesión cada vez. Véase <code>ADR-0038 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0038-a-named-theme-carries-the-board-palette-and-the-user-still-has-the-last-word.md&gt;</code>__.</p>
<p><em>Seguir el sistema</em> es el valor por defecto: obedece a la preferencia claro/oscuro del escritorio, incluso cuando cambia a mitad de sesión. Una herramienta no impone su claro o su oscuro a un escritorio que ya ha decidido.</p>
<p>La pestaña <em>Interfaz</em> permite también elegir el idioma entre inglés, francés, alemán, italiano, español, finés, japonés, griego y ruso. Toda la interfaz (barra de herramientas, paneles, mensajes, ayuda) se traduce al idioma seleccionado. La elección de idioma se guarda y se conserva de una sesión a otra.</p>
<p>La misma pestaña ofrece también el botón <strong>Compactar la base</strong>, que recupera el espacio en disco dejado por las eliminaciones (partidas, torneos, purgas): la base de datos nunca se reduce por sí sola cuando se borran datos, hay que pedir explícitamente esa compactación. La operación puede tardar en una base grande y necesita, temporalmente, alrededor del doble de su tamaño en espacio libre (blunderDB se niega a arrancar en lugar de arriesgar una compactación interrumpida); por eso se pide confirmación antes de lanzarla. El resultado — el espacio ganado, en megabytes — se muestra después en la barra de estado. La misma operación está disponible en línea de comandos mediante <code>blunderdb vacuum</code> (véase Interfaz de línea de comandos (CLI)).</p>
<p>El botón <strong>Abrir la carpeta de registros</strong>, justo debajo, abre la carpeta que contiene el registro de la aplicación — útil para adjuntar detalles a un informe de error, sobre todo cuando blunderDB se ha iniciado desde un acceso directo o un doble clic, sin terminal asociada que muestre nada.</p>
<p>La casilla <strong>Buscar actualizaciones al iniciar</strong>, desactivada por defecto, consulta una vez por arranque la página de versiones del repositorio de GitHub y muestra en la barra de estado un mensaje si hay una versión más reciente — nunca una ventana que impida trabajar. Esta comprobación queda automáticamente desactivada en una instalación hecha mediante un gestor de paquetes (Flatpak, Homebrew, un paquete de la distribución…): entonces es ese canal el que gestiona las actualizaciones, no blunderDB.</p>
<p>La pestaña <em>Colores del tablero</em> permite personalizar los colores del tablero. Cada elemento dispone de su propio selector de color: el fondo, el borde, las flechas claras y oscuras, las fichas del jugador 1 y del jugador 2, los dados, los puntos de los dados y el cubo de dobles. El botón <em>Restablecer</em> devuelve todos los colores predeterminados. Al igual que el idioma, los colores elegidos se conservan de una sesión a otra.</p>
<p>La pestaña <em>Bearoff</em> gestiona las tablas de bearoff del panel Eval (véase Panel Eval). <strong>No están ni incrustadas en el ejecutable ni se descargan</strong>: blunderDB las calcula en la máquina que las usa, y el resultado es idéntico byte a byte a lo que produce gnubg — la huella SHA-256 se verifica antes de aceptar una tabla.</p>
<p>Las dos tablas corrientes (TS-06-06 para el veredicto de cubo, OS-06 para el EPC) se calculan en el primer arranque, en segundo plano y sin preguntar: unos seis segundos en un núcleo, durante los cuales la aplicación se usa normalmente. El panel Eval solo lo menciona si se pone allí una posición que necesita una tabla que aún no está lista.</p>
<p>La pestaña muestra el dominio activo y su origen, el estado de la tabla de un lado que lee el EPC, la carpeta donde vive todo esto, y la lista de las tablas presentes con su tamaño y su veredicto. Cada fila se elimina individualmente, tras confirmación.</p>
<p><strong>Verificada o sin verificar.</strong> Una tabla <em>verificada</em> tiene exactamente los bytes que gnubg produce para su dominio: su huella SHA-256 figura en blunderDB y se ha vuelto a encontrar. Las huellas registradas para las tablas one-sided (OS-06 a OS-10) son las que produce la herramienta <code>makebearoff</code> de GNUbg 1.08. Una tabla <em>sin verificar</em> está bien formada pero su dominio no tiene huella registrada — no se le reprocha nada, simplemente nadie la ha comparado con la referencia. Una tabla <em>corrupta</em> se contradice a sí misma y nunca se lee; se recalcula.</p>
<p><strong>Calcular una tabla más amplia.</strong> El dominio se elige en una lista de dos familias, junto con el número de núcleos a dedicarle (por defecto todos menos uno, para que la máquina siga siendo utilizable):</p>
<ul>
<li><strong>cubo exacto (dos lados)</strong>, de TS-06-06 a TS-06-15: amplía el dominio donde la probabilidad de victoria y el veredicto de cubo se leen en lugar de estimarse;</li>
<li><strong>EPC fuera del cuadro (un lado)</strong>, de OS-06 a OS-10: amplía la distancia a la que una ficha puede estar sin que el bloque EPC se calle. Este barrido solo lee posiciones más pequeñas que la que calcula, así que es secuencial por construcción y el número de núcleos no le sirve de nada — el selector lo dice atenuándose.</li>
</ul>
<p>Antes de lanzar nada, la pestaña indica tres cifras para el dominio elegido: el tamaño en disco, la memoria necesaria durante el cálculo y el tiempo que debería tardar <em>en esta máquina</em>. Este último empieza como estimación y se vuelve medida: cada cálculo suficientemente amplio registra su propia velocidad y la conserva. Un dominio que la memoria disponible no permite se ofrece atenuado, con la razón — «harían falta 24 GB y quedan 12» es una respuesta, una fila ausente no lo sería.</p>
<p>Como orden de magnitud, en una máquina de dieciséis hilos: TS-06-09 pesa 191 MB y tarda una decena de segundos, TS-06-11 pesa 1,2 GB y unos minutos, TS-06-13 supera lo que la mayoría de las máquinas pueden mantener en memoria. Del lado de un lado, en un núcleo: OS-07 pesa 4,9 MB y tarda 17 s, OS-08 15 MB y 1 min 20, OS-10 117 MB y media hora.</p>
<p><strong>Pausa y reanudación.</strong> Durante el cálculo, el progreso muestra el tiempo restante <em>medido</em> y dos botones distintos: <em>Pausa</em> y <em>Cancelar</em>. La pausa escribe el estado del cálculo junto a la tabla; relanzarlo continúa donde se detuvo en lugar de empezar de nuevo. Cancelar no guarda nada. Cerrar la ventana de configuración no interrumpe nada — el cálculo continúa en segundo plano.</p>
<p>Un cálculo en pausa se reencuentra en el siguiente arranque, con su nombre y su cifra («TS-06-09 interrumpida al 43 %»), con <em>Reanudar</em> y <em>Eliminar</em>. Nada se reinicia solo: es el usuario quien pidió la parada.</p>
<p>La pestaña permite por último apuntar a un archivo <code>.bd</code> de dos lados externo, por ejemplo una base producida por el propio gnubg: gana la tabla con el dominio más amplio.</p>
<p>La pestaña <em>General</em> lleva por último <strong>Reparar los análisis</strong>: las columnas de análisis que consultan la búsqueda y las estadísticas son una proyección de los análisis almacenados, que quedan intactos. Un fallo de proyección se repara pues sin reimportar nada. Es explícito y nunca automático — reescribir las columnas de análisis de alguien por el mero hecho de que abra su base no es algo que una herramienta deba hacer a sus espaldas. El mismo <code>blunderdb repair</code> está disponible en la línea de comandos.</p>
<p>La pestaña <strong>gammonNet</strong> ajusta el evaluador integrado (véase <code>ADR-0011 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0011-gammonnet-is-ported-to-go-and-the-representation-boundary-sits-at-the-evaluator-s-edge.md&gt;</code>__). En ella se regulan dos profundidades de búsqueda, con nombre propio y guardadas por separado — bajar una nunca modifica la otra:</p>
<ul>
<li><strong>Profundidad de visualización</strong> — la comodidad interactiva durante la edición del tablero; nunca se escribe en la base.</li>
<li><strong>Profundidad de análisis</strong> — lo que el lote de análisis posterior a la importación escribe en el Análisis de una posición.</li>
</ul>
<p>Ambas valen por defecto <strong>2-ply</strong>, la configuración canónica. La pestaña ofrece también la <strong>poda</strong> (por defecto <code>k=12</code>) y el <strong>número de jugadas candidatas mostradas</strong> (por defecto 10), así como una casilla <strong>analizar automáticamente tras la importación</strong> que, una vez activada, comprueba después de cada importación si quedan posiciones <strong>sin ningún análisis</strong> (ni gammonNet, ni XG, ni GNUbg, ni BGBlitz — la regla es « una evaluación solo rellena un hueco », nunca un reemplazo) y, en su caso, lanza en segundo plano un análisis gammonNet a la profundidad de análisis configurada. Un botón <strong>Analizar ahora</strong> relanza manualmente la misma puesta al día, útil para una biblioteca creada antes de que existiera esta función.</p>
<p>Un segundo botón, <strong>Reanalizar posiciones obsoletas</strong>, cubre el caso contrario: una posición ya analizada por gammonNet, pero cuyo análisis almacenado fue escrito por una versión del motor más antigua que la que se ejecuta ahora, o a una profundidad distinta de la profundidad de análisis configurada arriba, se señala allí como obsoleta y se reevalúa. Una posición que además lleva un análisis de XG, GNUbg o BGBlitz nunca es tocada por este botón, sea cual sea su contenido de gammonNet — la protección de <code>ADR-0013 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0013-evaluations-fill-gaps-an-imported-analysis-is-never-overwritten.md&gt;</code>__ sigue siendo incondicional. El número mostrado junto a cada botón (posiciones sin análisis, posiciones obsoletas) es puramente informativo; el lote recalcula su propia lista al arrancar.</p>
<p>Ambos lotes son <strong>acotados, visibles y cancelables, nunca un demonio silencioso</strong>: su progreso (<code>posiciones analizadas / total</code>) y un botón de cancelación aparecen en la barra de estado durante toda su duración, y desaparecen una vez terminados en favor de un mensaje que resume el resultado — cuántas posiciones fueron <strong>analizadas</strong>, cuántas fueron <strong>rechazadas</strong> (una posición que gammonNet declina evaluar, como una puntuación de partida fuera del alcance de su tabla, lo cual nunca es un fallo) y cuántas <strong>fallaron</strong> (reintentadas, sin cambios, en la siguiente ejecución). Cerrar la aplicación durante uno u otro no pierde nada: cada posición analizada se escribe sobre la marcha, y la siguiente ejecución retoma exactamente donde el análisis se había detenido, sin ningún registro que llevar.</p>
<p><strong>Un partido importado sin análisis obtiene así un PR.</strong> Es el caso de un partido jugado en línea, o de un fichero Jellyfish <code>.mat</code>, que nadie ha pasado por XG: blunderDB conocía sus posiciones y las jugadas realizadas, pero ningún análisis decía cuánto valían. Una vez pasado el lote, la jugada realmente hecha se compara con la clasificación de gammonNet y la diferencia alimenta el PR, la tasa de error, las peores decisiones y todos los demás indicadores, exactamente como en un partido analizado por XG. La comparación no inventa nada: la jugada realizada viene de la tabla de jugadas del partido, escrita en la importación, llevara el fichero un análisis o no.</p>
<p>Una base analizada con una versión anterior a esta no necesita volver a evaluarse: <code>blunderdb repair</code> recalcula las columnas a partir de los análisis y las jugadas ya almacenados y devuelve su PR a esos partidos (véase repair).</p>
<p>Una reserva honesta: una posición se identifica por su estructura, de modo que una posición encontrada dos veces — bien jugada una vez, mal la otra — solo lleva una diferencia, la de su primera aparición registrada. No es propio de este cálculo: una biblioteca XG tiene exactamente la misma forma.</p>
<h4>Carpeta vigilada</h4>
<p>La pestaña <strong>Carpeta vigilada</strong> pide a blunderDB que mire una carpeta mientras se ejecuta e importe cada fichero de partido que <strong>aparezca</strong> en ella. Jugar una sesión en eXtreme Gammon, volver a blunderDB, y encontrar los partidos ya ahí.</p>
<p>Nada se adivina. Mientras no se designe una carpeta no hay vigilancia: blunderDB no empieza a leer un directorio porque haya supuesto dónde viven sus partidos. El botón <strong>Proponer</strong> mira los lugares habituales de esta máquina y solo propone uno si existe de verdad; si no, lo dice, y designar la carpeta le corresponde a usted.</p>
<p>Tres puntos merecen conocerse antes de marcar la casilla:</p>
<ul>
<li><strong>Solo se importan los ficheros que aparecen.</strong> Lo que la carpeta ya contiene cuando arranca la vigilancia se registra como conocido y se deja en paz: apuntar una vigilancia a cuatro años de partidos no debe importarlos todos. Para importar lo que hay, use la importación de carpeta, que existe para eso — y ambas se componen muy bien, la importación primero, la vigilancia después.</li>
<li><strong>Un fichero se importa solo cuando su tamaño se ha estabilizado.</strong> Un partido que otro programa está escribiendo crece de un vistazo a otro; importarlo a medio escribir daría un error de análisis sobre el que nadie puede actuar. blunderDB espera, pues, a ver dos veces el mismo fichero sin cambios.</li>
<li><strong>La importación es silenciosa.</strong> Estaba estudiando una posición cuando llegaron sus partidos: quitarle la pantalla sería el peor momento. La importación se hace sin ventana, y la barra de estado muestra una franja con el recuento de partidos importados, ignorados (duplicados) y fallidos, con un botón que abre el informe completo si lo desea. Todo lo demás es idéntico a una importación manual: mismos duplicados detectados, mismo lote de importación, mismo análisis automático si está activado.</li>
</ul>
<p>El intervalo por defecto es de diez segundos; el mínimo es de dos. La carpeta no se recorre recursivamente: una carpeta vigilada es el sitio donde una herramienta deposita sus partidos, no un árbol que explorar. Un recurso de red desmontado no detiene la vigilancia ni hace que su contenido pase por nuevo a su regreso.</p>
<p>La misma vigilancia existe en línea de comandos, con <code>blunderdb import --type batch --dir &lt;carpeta&gt; --watch</code> (véase Interfaz de línea de comandos (CLI)): es la forma que puede usar un servidor, una tarea programada o un script.</p>
<p>La ventana de configuración también incluye ajustes de visualización de la interfaz. Un control deslizante de <strong>escala de la interfaz</strong> permite agrandar o reducir todos los elementos, lo que resulta útil en pantallas de alta densidad o para mejorar la legibilidad. Un menú <strong>posición de los paneles</strong> determina la ubicación de los paneles (búsqueda, partidas, análisis) con respecto al tablero: <em>abajo</em>, <em>al lado</em> o <em>automática</em> (en este caso el lado se elige en las pantallas anchas para aprovechar mejor el espacio disponible). Como los demás ajustes, estas opciones se conservan de una sesión a otra.</p>
<h3>Visitas guiadas y base de ejemplo</h3>
<p>Para facilitar la toma de contacto, blunderDB ofrece <strong>visitas guiadas</strong> de la interfaz. El catálogo de visitas se abre desde la barra de herramientas o con el comando <code>tour</code> (alias <code>tutorial</code>). Hay siete visitas disponibles: una visita general de la interfaz, y visitas dedicadas a la búsqueda de posiciones, a la revisión de partidas, a la revisión de torneos, al panel Eval, al repaso Anki y a las estadísticas. Cada visita resalta los elementos correspondientes de la interfaz, paso a paso, abre de paso el panel del que habla, y puede repetirse en cualquier momento. En el primer arranque, la visita general se ofrece automáticamente.</p>
<p>El comando <code>demo</code> carga una <strong>base de ejemplo</strong> que permite descubrir las funcionalidades de la herramienta sin importar las propias partidas: tres partidas (dos de ellas agrupadas en un torneo) analizadas por eXtreme Gammon, BGBlitz y gammonNet, tres colecciones temáticas, comentarios etiquetados (<code>#blunder</code>, <code>#cube</code>) y un mazo Anki con su registro de repasos. Los jugadores, el torneo y el lugar son ficticios. Las visitas guiadas se apoyan en esta base cuando no hay ninguna base abierta.</p>
<h3>Navegación por las posiciones</h3>
<p>Por defecto, blunderDB permite:</p>
<ul>
<li>recorrer las distintas posiciones de la biblioteca actual — que nunca se carga de una vez: blunderDB solo mantiene la lista de identificadores y carga las posiciones por ventanas de cincuenta alrededor de la que se muestra, de modo que una base de varias decenas de miles de posiciones se abre tan rápido como una pequeña,</li>
<li>mostrar la información de análisis asociada a una posición,</li>
<li>mostrar, añadir y modificar los comentarios de una posición.</li>
</ul>
<p>El botón <strong>Ir a la posición</strong> de la barra de herramientas abre una ventana donde escribir directamente el índice de una posición para saltar a ella, sin tener que desplazarse. Es el equivalente gráfico del comando <code>[number]</code> de la línea de comandos (véase Posiciones y navegación).</p>
<div class="admonition tip">
<p>Consulte Atajos de teclado para ver los atajos disponibles.</p>
</div>
<h3>Edición de posiciones</h3>
<p>Pulsar la tecla <em>TAB</em> abre el panel de búsqueda y permite editar una posición en el tablero para añadirla a la base de datos o para definir una estructura de posición que buscar. La distribución de las fichas, el cubo, la puntuación y el turno pueden modificarse con el ratón (véase Editar una posición).</p>
<div class="admonition tip">
<p>Consulte Atajos de teclado para ver los atajos disponibles.</p>
</div>
<h3>La línea de comandos</h3>
<p>La línea de comandos, integrada en la barra de estado, permite realizar todas las funcionalidades de blunderDB disponibles en la interfaz gráfica: operaciones generales sobre la base de datos, navegación por las posiciones, visualización del análisis o de los comentarios, búsqueda de posiciones según filtros... Tras una primera toma de contacto con la interfaz, se recomienda utilizar progresivamente la línea de comandos, que permite un uso potente y fluido de blunderDB, especialmente para las funcionalidades de búsqueda de posiciones.</p>
<p>Para abrir la línea de comandos, pulse la tecla <em>ESPACIO</em>. Para enviar una consulta y cerrar la línea de comandos, pulse la tecla <em>INTRO</em>.</p>
<p>blunderDB ejecuta las consultas enviadas por el usuario siempre que sean válidas y modifica inmediatamente el estado de la base de datos si procede. No se requieren acciones de guardado explícitas por parte del usuario.</p>
<div class="admonition tip">
<p>Consulte la lista de comandos para ver la lista de comandos disponibles en la línea de comandos.</p>
</div>
<h3>Panel de Análisis</h3>
<p>El panel <strong>Análisis</strong> (<em>CTRL-L</em>) muestra los datos de análisis de la posición actual importados desde eXtreme Gammon (XG), GNUbg o BGBlitz. Presenta las mejores alternativas (jugadas de fichas o decisiones de cubo) con sus valores de equidad y los errores correspondientes. La tecla <em>d</em> alterna entre el análisis de las jugadas de fichas y el análisis del cubo. Durante la navegación por una partida, la jugada realmente jugada se resalta en la lista de alternativas. Pulse <em>CTRL-L</em> o ejecute el comando <code>list</code> para mostrar u ocultar el panel.</p>
<p>Bajo las tablas, una <strong>frase</strong> dice a veces lo que costó la decisión jugada y por qué: «Pierde 120 mMWC: la jugada realizada deja tres fichas sueltas donde 13/7 8/7 solo deja una.» Procede de seis reglas medibles — la exposición, un punto de la casa hecho o perdido, las posibilidades de gammon abandonadas, una seguridad que cuesta más de lo que aporta, y los dos sentidos de un error de cubo (doblar demasiado tarde o demasiado pronto, aceptar demasiado amplio o pasar demasiado estricto).</p>
<p>La regla que importa es la del <strong>silencio</strong>: la frase solo aparece cuando una regla se aplica con confianza, y sobre un error que supera el umbral a partir del cual los motores coinciden en que lo es. El resto del tiempo no hay frase — ni marco vacío, ni «no lo sabemos». Una explicación equivocada cuesta más que ninguna: enseña algo inexacto.</p>
<p>Cuando una posición ha sido juzgada por <strong>varios motores</strong>, una franja en la cabecera del panel los pone lado a lado: una línea por motor, con su profundidad y su respuesta — el veredicto de cubo, o su propia mejor jugada. Dice primero si coinciden, y es la discrepancia lo que la justifica: «XG dice doblar, tomar; gammonNet dice no doblar» se lee de un vistazo, donde había que comparar dos tablas en diagonal.</p>
<p>La mejor jugada de un motor es la mejor <strong>de ese motor</strong>: la lista de jugadas candidatas está ordenada por equidad, con todos los motores mezclados, así que su primer elemento no es la mejor jugada de ninguno en particular.</p>
<p>La franja aparece solo si de verdad hay varios motores, y existe únicamente en este panel: el panel Eval presenta <strong>una</strong> decisión, la del motor integrado (<code>ADR-0017 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0017-the-panel-shows-position-facts-plus-the-one-decision-the-board-asks.md&gt;</code>__), y una comparación no tendría sitio allí.</p>
<p>Las jugadas se escriben como se leen en el tablero, aquí igual que en el panel Eval: la ficha menos avanzada se mueve primero, y <strong>una ficha que encadena varios dados se escribe una sola vez</strong> — un 64 jugado con la misma ficha se lee <code>24/14</code>, y <code>24/14*</code> si golpea al llegar. El detalle del encadenamiento solo reaparece cuando dice algo más: un golpe <em>por el camino</em> conserva su punto de paso, <code>24/18* 18/14</code>, sin lo cual el golpe en el 18 desaparecería de la notación.</p>
<p>La equidad de un análisis importado sigue la misma regla que el panel Eval: la columna indica su propio marco de referencia, «Equity (money)» o «Equity (match)» según el marcador de la posición analizada, nunca un simple «Equity» mudo sobre la escala. Las reglas <strong>Jacoby</strong> y <strong>Beaver</strong> activas en una posición de money game también se muestran, en insignias bajo la tabla de decisión del cubo.</p>
<h3>Panel de Comentarios</h3>
<p>El panel <strong>Comentarios</strong> (<em>CTRL-P</em>) muestra, añade y edita los comentarios asociados a la posición actual. Una posición puede llevar varios: se muestran todos, del más reciente al más antiguo. Los comentarios importados de archivos XG se asocian automáticamente a las posiciones correspondientes. Pulse <em>CTRL-P</em> o ejecute el comando <code>comment</code> para mostrar u ocultar el panel.</p>
<p>Cada comentario procedente de un archivo lleva una <strong>etiqueta de procedencia</strong> (<code>XG</code>, <code>GNU BG</code>, <code>BGF</code>, o <em>importado</em> cuando la procedencia nunca se registró). Los comentarios que usted escribió no llevan ninguna: es el caso corriente, y señalarlo en cada línea sería ruido. Modificar un comentario importado se lo atribuye: tras la modificación, la frase es suya.</p>
<p>Esta distinción se nota en otro sitio: borrar una partida ya no destruye una posición sobre la que <strong>usted</strong> había escrito. Una nota tomada del archivo de origen sí desaparece con la partida que la trajo.</p>
<h4>Las etiquetas</h4>
<p>Una <strong>etiqueta</strong> es una <code>#palabra</code> escrita en un comentario. Nada la declara, ninguna tabla la contiene, y es deliberado: el vocabulario es su propia prosa, y exigir una declaración antes de poder etiquetar convertiría un hábito en papeleo.</p>
<p>Lo que faltaba era la otra mitad: <strong>ver</strong> el vocabulario que uno se ha construido, y hacer clic en una etiqueta en vez de recordar cómo se escribía. El comando <code>tags</code>, o el botón <code>#</code> junto al campo de escritura, abre la ventana del vocabulario: las etiquetas de esta base, cada una con el <strong>número de posiciones</strong> que la llevan, pulsables para lanzar la búsqueda correspondiente. Bajo la lista figuran las etiquetas recomendadas que la base aún no usa — un vocabulario tomado de la literatura del backgammon (<code>#blitz</code>, <code>#prime</code>, <code>#holding</code>, <code>#backgame</code>, <code>#containment</code>, <code>#crunch</code>, <code>#ace-point</code>, <code>#timing</code>…), sugerido y nunca impuesto: una etiqueta ausente de esa lista vale exactamente lo mismo que una que figure en ella.</p>
<p>Al escribir, un <code>#</code> propone las etiquetas que <strong>esta base</strong> ya usa, y luego las recomendadas. Es lo que evita escribir <code>#back-game</code> un día y <code>#backgame</code> al siguiente, algo que nada más detectaría.</p>
<p>La búsqueda por etiqueta se escribe <code>#prime</code> en la línea de comandos. Es <strong>delimitada</strong>: <code>#prime</code> no encuentra <code>#priming</code>, mientras que una búsqueda de texto ordinaria, que busca una subcadena, no sabe distinguirlas. Varias etiquetas se <strong>acumulan</strong> — <code>s #prime #backgame</code> pide las posiciones que llevan ambas — porque una posición lleva varias etiquetas: nombrar dos solo puede querer decir «las dos». Es lo contrario del filtro de fase o de procedencia, donde una posición solo tiene un valor y nombrar dos valores solo puede querer decir «una u otra».</p>
<p>La misma lista se obtiene fuera de la interfaz con <code>blunderdb list --type tags</code> (véase Interfaz de línea de comandos (CLI)).</p>
<h3>La papelera</h3>
<p>Eliminar una posición, una colección o un comentario pasa ahora por una <strong>papelera</strong>: la eliminación se produce, pero una copia de lo que desaparece se conserva treinta días. El comando <code>trash</code> abre la ventana que las lista, cada una con <em>Restaurar</em> y <em>Eliminar</em>.</p>
<p>Una posición restaurada vuelve con <strong>su análisis y sus comentarios</strong> — devolverla desnuda sería una restauración solo de nombre. No vuelve con su número antiguo: la fila original ya no existe, y blunderDB la vuelve a guardar por su huella, lo que garantiza que nunca crea un duplicado pero le da un identificador nuevo. Una colección vuelve con su lista; las posiciones que contenía nunca fueron eliminadas — una colección es una vista sobre ellas.</p>
<p>Lo que tiene más de treinta días lo elimina el comando <code>vacuum</code>, nunca la apertura de una base: no hacer <code>vacuum</code> es conservarlo todo.</p>
<div class="admonition note">
<p>La papelera no viaja. Una exportación no la lleva, y eliminar una partida no pone nada en ella: la purga de posiciones huérfanas que sigue a la eliminación de una partida es una limpieza automática, no un gesto del usuario — véase la regla de retención en Panel de Partidas.</p>
</div>
<h3>Panel de Búsqueda</h3>
<p>El panel <strong>Búsqueda</strong> (<em>CTRL-F</em> o <em>TAB</em>) permite filtrar las posiciones según criterios libremente combinables: estructura de fichas, tipo de decisión de cubo, magnitud del error, fechas, etiquetas, etc. La tecla <em>TAB</em> abre simultáneamente el panel de búsqueda y el editor de posición, lo que permite definir una estructura de fichas que buscar directamente en el tablero.</p>
<p>Para afinar una búsqueda entre las posiciones filtradas actualmente, use el comando <code>ss</code> seguido de filtros (p. ej.: <code>ss nc</code>, <code>ss E&gt;40</code>). El panel de búsqueda ofrece también una casilla <em>Buscar en los resultados actuales</em> para la misma funcionalidad.</p>
<p>El panel ofrece un control explícito del <strong>tipo de decisión</strong> buscado: <em>Indiferente</em> (ningún filtro), <em>Fichas</em> (decisiones de jugada) o <em>Cubo</em> (decisiones de cubo). Cuando se selecciona <em>Cubo</em>, una segunda lista precisa el subtipo: <em>Todos</em>, <em>Doblar / No doblar</em> (el jugador con el turno debe decidir si doblar) o <em>Aceptar / Pasar</em> (respuesta a un doblaje del rival). El control está sincronizado con el tablero: modificar los dados o el cubo en el tablero actualiza el tipo de decisión, y viceversa. En modo <em>Aceptar / Pasar</em>, el cubo se muestra en el centro del tablero con el valor ofrecido; ese valor sigue siendo editable.</p>
<p>La <strong>fase de la partida</strong> — apertura, medio juego, carrera, retirada de fichas — es una etiqueta que blunderDB calcula únicamente a partir del tablero. Nunca es editable y se puede buscar con el token <code>ph:</code> de la línea de comandos (<code>ph:race</code>, repetible: <code>ph:race ph:bearoff</code>). Tres de sus cuatro fronteras son las que GNU Backgammon emplea para dirigir sus redes; la cuarta, donde termina la apertura, es una convención de blunderDB: una posición sigue en la apertura mientras ninguno de los dos bandos haya movido más de cuatro fichas de sus puntos de partida, no se haya retirado ninguna y ninguna esté en la barra.</p>
<div class="admonition note">
<p>La etiqueta la recalcula el comando <code>blunderdb repair</code>. En una base abierta por primera vez con esta versión, el cálculo se hace una vez, al abrirla. Una base cuyas fases nunca se han calculado no devuelve nada para <code>ph:</code> — nada, en lugar de una respuesta falsa.</p>
</div>
<p>El comando <code>like</code> responde a una pregunta distinta de la de los tokens: sustituye la lista recorrida por las posiciones más <strong>cercanas</strong> a la actual, de la más cercana a la más lejana. La cercanía es una distancia de transporte, expresada en pips de ficha — la cantidad de movimiento de fichas que separa ambas posiciones — y el punto de vista es siempre el del jugador que mueve. No es un filtro: la similitud <strong>ordena</strong> toda la biblioteca en vez de restringirla, y por tanto no se combina con los tokens.</p>
<p>El token <code>n</code> cuenta <strong>encuentros</strong>: <code>n&gt;3</code> conserva las posiciones a las que llegan más de tres jugadas, en todos los partidos. Es otra pregunta distinta de «qué he fallado» — una posición encontrada veinte veces y bien jugada diecinueve sigue siendo la que hay que saber de memoria. Se cuentan las jugadas, no los partidos: la misma posición dos veces en un partido cuenta dos, porque fueron dos decisiones.</p>
<p>Una frase en palabras puede sustituir a los tokens, con el comando <code>ask</code>: <code>ask my cube blunders at a score</code>. La frase se <strong>traduce a tokens</strong>, escritos en la barra de comandos — se releen y luego se lanza. Nada se adivina y nada sale de la máquina: el vocabulario es fijo, la misma frase da siempre la misma consulta, y lo que no se ha entendido se <strong>dice</strong> en lugar de pasarse por alto. Una traducción equivocada se ve así antes de devolver resultados equivocados, y los tokens se aprenden leyéndolos.</p>
<p>Dos intenciones no son tokens y se ponen en el tablero de búsqueda en vez de en la línea: «cubo» o «fichas» (el tipo de decisión) y «al marcador» o «money». <code>ask</code> las pone allí.</p>
<p>El <strong>plan de juego</strong> es una segunda etiqueta derivada, junto a la fase, y responde a la pregunta que un paquete de filtros guardados no sabe plantear: «muéstrame mis errores en holding game». Token <code>gt:</code>, repetible (<code>gt:holding gt:mutualholding</code>), desde el punto de vista del <strong>jugador que mueve</strong> — el plan en el que se tomaba la decisión.</p>
<p>Los diez planes reconocidos, en el orden en que las reglas los agotan, del más específico al más general:</p>
<ul>
<li><code>race</code> — las fichas más atrasadas de ambos bandos se han cruzado: ya no es posible ningún contacto. Frontera de GNU Backgammon.</li>
<li><code>bearin</code> — el jugador que mueve está entrando las fichas mientras el adversario aún mantiene un ancla en su casa.</li>
<li><code>crunch</code> — el jugador que mueve tiene como mucho seis fichas fuera de sus puntos 1 y 2. Regla de GNU Backgammon, umbral de su autor.</li>
<li><code>backgame</code> — dos o más anclas en la casa del adversario.</li>
<li><code>acepoint</code> — una sola ancla, en el punto uno del adversario, con al menos veinte pips de retraso.</li>
<li><code>blitz</code> — tres o más puntos de la casa hechos, y el adversario en la barra o con una ficha suelta que golpear en esa casa.</li>
<li><code>primevprime</code> — ambos bandos mantienen un bloqueo de al menos cuatro puntos, y cada uno tiene una ficha atrapada tras el del otro.</li>
<li><code>mutualholding</code> — ambos bandos mantienen un ancla alta.</li>
<li><code>holding</code> — el jugador que mueve mantiene un ancla alta, el adversario no.</li>
<li><code>contact</code> — contacto, y ninguno de los planes anteriores. La apertura acaba aquí.</li>
</ul>
<p>Tres de estas reglas son las de GNU Backgammon y están documentadas; las demás son <strong>convenciones de blunderDB</strong>. La literatura del backgammon describe los planes de juego sin cifrar sus fronteras, y no se ha publicado ninguna medida de acuerdo entre clasificadores para este problema. Los umbrales no documentados — tres puntos de casa para un blitz, cuatro puntos para un bloqueo, veinte pips de retraso para un ace-point game — se enuncian por tanto aquí en vez de esconderse en el código, y están versionados: cámbielos, ejecute <code>blunderdb repair</code>, y toda la base se reetiqueta.</p>
<div class="admonition note">
<p>Se conserva una sola etiqueta por posición, la del jugador que mueve. Una etiqueta derivada nunca es editable, nunca se exporta como verdad, y una base cuyos planes nunca se han calculado no devuelve nada para <code>gt:</code> — igual que para <code>ph:</code>.</p>
</div>
<p>El filtro <strong>Marcada</strong> conserva las posiciones que ha marcado en el programa de origen de la partida. Solo eXtreme Gammon produce esta información, registrada jugada a jugada en el archivo <code>.xg</code>; blunderDB la lee al importar y la conserva. Una decisión de cubo marcada da dos posiciones marcadas, el doblez y la aceptación/paso, ya que blunderDB divide en dos lo que el archivo de origen registra como una sola decisión.</p>
<div class="admonition note">
<p>El marcado no es retroactivo: las partidas ya presentes en la base no contienen esta información, puesto que solo existe en los archivos de origen. Basta con volver a importar el archivo <code>.xg</code> correspondiente — la importación detecta el duplicado y no añade nada más que las marcas, sin tocar los comentarios ni los análisis existentes. El marcado no puede ponerse ni quitarse desde blunderDB: para una lista de trabajo temporal, utilice más bien una colección.</p>
</div>
<p>El filtro <strong>Comentario</strong> consulta los comentarios asociados a las posiciones según tres modos exclusivos. <em>contiene el texto</em> busca una o varias palabras en el texto de los comentarios (campo de entrada, palabras separadas por <code>;</code>, al menos una debe coincidir); <em>tiene un comentario</em> conserva toda posición que lleve un comentario, sea cual sea su contenido; <em>sin comentario</em> conserva por el contrario las posiciones no anotadas — útil, combinado con un filtro de error o de fecha, para elaborar la lista de lo que queda por comentar.</p>
<div class="admonition note">
<p>Los comentarios importados de un archivo de partida (XG, GNUbg) cuentan como comentarios. Para quedarse solo con los suyos, añada el token <code>co:user</code> en la línea de comandos (<code>co:xg</code>, <code>co:gnubg</code>, <code>co:bgf</code> y <code>co:unknown</code> designan las demás procedencias). Además, los comentarios asociados a una <em>partida</em> o a un <em>torneo</em> no se ven afectados: anotan la partida o el torneo, no sus posiciones.</p>
</div>
<p>El filtro <strong>Partidas y Torneos</strong> se apoya en un selector común (ventana modal) en lugar de la introducción de identificadores numéricos: dos listas de casillas, una para las partidas y otra para los torneos, cada una filtrable por texto (jugador, fecha, evento para las partidas; nombre, fecha, lugar para los torneos), con botones <em>Todo</em> / <em>Ninguno</em> que solo actúan sobre el subconjunto actualmente filtrado. Marcar un torneo marca automáticamente (y atenúa) sus partidas miembro en la lista de partidas, dejando patente que un torneo equivale al conjunto de sus partidas.</p>
<p>El panel de búsqueda cuenta con tres pestañas en su borde izquierdo: <em>Búsqueda</em> (los filtros), <em>Historial</em> y <em>Guardados</em>. La pestaña <strong>Historial</strong> enumera las búsquedas anteriores con su fecha y su comando: un clic selecciona una búsqueda y muestra la posición asociada en el tablero, un doble clic la vuelve a ejecutar. Cada entrada puede guardarse en la biblioteca de filtros (icono de marcador, dándole un nombre al filtro) o eliminarse. La pestaña <strong>Guardados</strong> contiene la <strong>biblioteca de filtros</strong>: hacer doble clic en un filtro guardado para relanzar la búsqueda correspondiente (véase Anexo: Uso avanzado de los filtros). El comando <code>history</code> (alias <code>hi</code>) abre el panel de búsqueda.</p>
<div class="admonition tip">
<p>Consulte la lista de comandos para ver la lista de filtros disponibles.</p>
</div>
<h3>Panel de Colecciones</h3>
<p>El panel <strong>Colecciones</strong> (<em>CTRL-B</em>) permite gestionar colecciones de posiciones. Las colecciones pueden crearse, renombrarse y eliminarse. Se les pueden añadir o quitar posiciones (tecla <em>Supr</em>, se pide confirmación). Haga doble clic en una colección para recorrer sus posiciones con las teclas <em>IZQUIERDA</em> y <em>DERECHA</em>. El orden de las colecciones y de las posiciones dentro de una colección puede cambiarse arrastrando y soltando. Pulse <em>CTRL-B</em> o ejecute el comando <code>collection</code> para mostrar u ocultar el panel.</p>
<h3>Importación: lo que se escribe, lo que nunca se escribe</h3>
<p>Importar un match, una posición u otra base añade lo que falta; no reemplaza lo que ya está ahí.</p>
<ul>
<li><strong>Una posición nunca se duplica.</strong> Es su identidad — fichas, cubo, dados, marcador — la que la reconoce, nunca el archivo del que proviene: la misma posición encontrada en dos partidas sigue siendo una sola fila.</li>
<li><strong>Un análisis por motor.</strong> eXtreme Gammon, GNUbg, BGBlitz y el evaluador integrado conviven en una misma posición, y el panel Análisis indica el origen de cada uno. Importar uno no borra el otro.</li>
<li><strong>Un análisis importado nunca se recalcula.</strong> blunderDB lo guarda tal cual, con su etiqueta de nivel («3-ply», «XG Roller++», «Book»), sus equidades, sus errores, sus probabilidades y la suerte de la tirada. La regla es «una evaluación solo rellena un hueco»: el análisis automático tras la importación solo visita las posiciones sin <strong>ningún</strong> análisis, y <em>Reanalizar posiciones obsoletas</em> deja intacta toda posición que lleve un análisis importado (véase Configuración).</li>
<li><strong>Reimportar el mismo archivo no reescribe nada.</strong> El match se reconoce como ya presente; solo se añaden las marcas puestas en el software de origen, sin tocar los comentarios ni los análisis.</li>
<li><strong>Lo que blunderDB nunca escribe</strong>: una suerte recalculada — se lee del archivo fuente, o queda desconocida — y un rollout, cuyos datos no abre dentro de un archivo <code>.xg</code> y que no sabe producir.</li>
</ul>
<p>Una colección puede estar <strong>viva</strong>: su contenido ya no es una lista hecha a mano sino el resultado de una <strong>búsqueda</strong>, reevaluado cada vez que se abre. El botón ◇ en la cabecera de la colección la hace viva con la última búsqueda lanzada; ◈ indica que ya lo está, y el mismo botón le devuelve su lista. Nada se destruye al hacerla viva: las posiciones que contenía siguen ahí al volver atrás.</p>
<p>Una colección viva cuya consulta lleva un token que esta versión ya no conoce <strong>se niega a abrirse</strong> y lo dice, en vez de devolver toda la base. Es el único fallo que un filtro guardado no debe tener: ensancharse en silencio.</p>
<h3>Panel de Partidas</h3>
<p>El panel <strong>Partidas</strong> (<em>CTRL-Tab</em>) lista las partidas importadas. Haga doble clic en una partida (o pulse <em>INTRO</em>) para navegar por sus jugadas. El comando <code>m</code> reanuda la navegación en la última partida visitada.</p>
<p>El usuario puede:</p>
<ul>
<li>recorrer las jugadas de una partida con las teclas <em>IZQUIERDA</em> y <em>DERECHA</em>,</li>
<li>pasar de una partida a otra con las teclas <em>PageUp</em> y <em>PageDown</em>,</li>
<li>mostrar el análisis de las jugadas (fichas y cubo) pulsando <em>CTRL-L</em>,</li>
<li>alternar entre el análisis de las jugadas de fichas y del cubo con la tecla <em>d</em>,</li>
<li>ver la jugada realmente jugada resaltada en el análisis.</li>
</ul>
<p>La última posición visitada en cada partida se guarda y se restaura automáticamente. Pulse <em>CTRL-Tab</em> o ejecute el comando <code>match</code> para mostrar u ocultar el panel.</p>
<p>El botón <strong>⊕</strong> de una fila enriquece ese partido desde un fichero. No hay nada nuevo detrás: reimportar el mismo partido en otro formato ya lo enriquece en su sitio — la huella canónica reconoce que se trata del mismo partido, y los análisis y comentarios del segundo fichero completan el primero. Lo que aporta el botón es que se encuentra: nadie adivina que una importación es también un enriquecimiento. El informe que sigue dice cuál de los dos ha ocurrido — «enriquecidos: 1» en lugar de «importados: 1».</p>
<p>Cada partida puede exportarse en transcripción Jellyfish <code>.mat</code> mediante el botón ⬇ de la lista de partidas o el botón <em>.mat</em> de la ficha de la partida.</p>
<p>El botón <strong>Fusionar jugadores</strong> de la barra de herramientas del panel abre una ventana que enumera todos los nombres de jugadores de la base con su número de partidas: seleccionar las variantes de ortografía de un mismo jugador, elegir el nombre canónico que se desea conservar y, a continuación, fusionar. Útil para unificar las estadísticas por jugador cuando un mismo jugador aparece con varios nombres.</p>
<p>Cuando una partida está abierta, aparece una <strong>barra de información</strong> sobre el tablero: recuerda los jugadores presentes (<em>jugador 1</em> contra <em>jugador 2</em>) así como el contexto de la partida (evento, lugar, ronda, fecha y longitud de la partida, cuando esa información está disponible). Esta barra también se muestra fuera del modo partida: cuando una posición estudiada (procedente de una búsqueda, de una colección o de un acceso directo) proviene de una o varias partidas, indica su <strong>procedencia</strong> — la primera partida implicada y, en su caso, una insignia « +N » que enumera las demás al pasar el cursor. Una posición importada por separado, que ninguna partida referencia, no muestra nada.</p>
<p>Al abrir una base que contiene partidas, el panel <strong>Partidas</strong> se muestra de inmediato y la revisión comienza directamente en la primera posición, para empezar a navegar de inmediato.</p>
<div class="admonition note">
<p>Una base de datos solo puede abrirse en escritura por una única ventana a la vez. Si abre una base ya abierta en otra ventana de blunderDB, se abre en modo de <strong>solo lectura</strong>: la navegación, la búsqueda y el análisis siguen siendo posibles, pero toda modificación queda desactivada y la barra de título muestra « [solo lectura] ».</p>
</div>
<div class="admonition tip">
<p>Consulte Atajos de teclado para ver los atajos disponibles.</p>
</div>
<h3>Panel de Torneos</h3>
<p>El panel <strong>Torneos</strong> (<em>CTRL-Y</em>) permite agrupar partidas en torneos para un seguimiento organizado y un análisis estadístico por evento. Los torneos pueden crearse, renombrarse y eliminarse; las partidas pueden asignarse a ellos. Las estadísticas del panel Stats pueden filtrarse por torneo. Pulse <em>CTRL-Y</em> para mostrar u ocultar el panel.</p>
<p>Los torneos se llenan solos al importar. Los archivos XG, GnuBG y BGF nombran su evento; al importar una partida nueva, blunderDB la clasifica en el torneo de ese nombre y lo crea si aún no existe. La fecha y el lugar del torneo quedan vacíos: es aquí donde se rellenan. Una partida ya presente en la base nunca se reclasifica: reimportar su archivo no deshace la organización hecha a mano.</p>
<p>La columna <strong>PR</strong> de cada torneo muestra el PR del <strong>jugador de referencia</strong> — es decir, el jugador presente en el mayor número de partidas del torneo (en caso de empate, el que haya tomado más decisiones). El PR no mezcla por tanto su juego con el de sus adversarios: para sus propios torneos, refleja únicamente su rendimiento. El nombre del jugador de referencia aparece en un cuadro emergente al pasar por encima del valor.</p>
<h3>Panel Stats</h3>
<h4>Introducción</h4>
<p>El panel <strong>Stats</strong> permite analizar el nivel de juego y seguir la progresión a lo largo del tiempo a partir de las posiciones importadas en la base de datos. Calcula y muestra los indicadores <strong>PR</strong> (<em>Performance Rating</em>) y <strong>MWC cost</strong> (Match Winning Chance cost) para el conjunto de las posiciones o para un subconjunto filtrado.</p>
<p>El panel Stats resulta especialmente útil para:</p>
<ul>
<li><strong>situar su nivel</strong> respecto a las bandas de nivel (<em>Clase mundial</em>, <em>Experto</em>, *Avanzado*…) gracias al PR global;</li>
<li><strong>seguir su progresión</strong> torneo a torneo o partida a partida gracias a los gráficos de la pestaña Progresión;</li>
<li><strong>identificar sus puntos débiles</strong>: la pestaña Errores para ver el reparto entre jugadas de fichas y decisiones de cubo, y la distribución de las magnitudes de error;</li>
<li><strong>comparar entre sí a los jugadores de la base</strong>, una fila por jugador, gracias a la pestaña Jugadores — útil para seguir una competición entera;</li>
<li><strong>acceder directamente a las posiciones implicadas</strong> haciendo clic en cualquier indicador (drill-down).</li>
</ul>
<h4>Apertura del panel</h4>
<p>Para abrir el panel Stats:</p>
<ul>
<li>Pulse <em>CTRL-D</em>.</li>
<li>Escriba el comando <code>stats</code> o <code>st</code> en la línea de comandos.</li>
</ul>
<div class="admonition note">
<p>El panel se actualiza automáticamente cada vez que se modifica el filtro. No recalcula las estadísticas al alternar simplemente entre PR ↔ MWC: ambas métricas se calculan simultáneamente por el backend.</p>
</div>
<h4>Barra de filtro</h4>
<p>La barra de filtro, en la parte superior del panel, permite restringir el cálculo a un subconjunto de posiciones.</p>
<h5>Perspectiva del jugador</h5>
<p>La lista desplegable <strong>Jugador</strong> permite filtrar las estadísticas según el jugador analizado. blunderDB selecciona automáticamente el jugador cuyo nombre aparece con más frecuencia en la base de datos, modificable en cualquier momento.</p>
<div class="admonition tip">
<p>Cambiar de jugador no provoca ninguna pérdida de datos; basta con volver a seleccionar el jugador anterior en la lista.</p>
</div>
<h5>Filtros disponibles</h5>
<ul>
<li><strong>Torneo(s)</strong> — restricción a uno o varios torneos. Pueden seleccionarse varios torneos simultáneamente.</li>
<li><strong>Fechas</strong> — intervalo temporal (<em>Desde</em> … <em>Hasta</em>). Si solo se indica la fecha de inicio, se incluyen las posiciones más recientes.</li>
<li><strong>Tipo de decisión</strong> — Todas / Jugadas de fichas / Decisiones de cubo.</li>
<li><strong>Longitud de match</strong> — restricción a longitudes de match concretas (1, 3, 5, 7, 9, 11, 13, 15, 21 puntos). Pueden combinarse varias longitudes.</li>
</ul>
<p>Un botón <strong>Reset</strong> restablece todos los filtros (salvo el jugador autodetectado).</p>
<div class="admonition note">
<p>Los filtros se guardan en la configuración de blunderDB (<code>config.yaml</code>) y se restauran en el próximo inicio.</p>
</div>
<h4>Conmutador PR / MWC</h4>
<p>El botón <strong>PR / MWC</strong> situado en la parte superior del panel alterna la métrica mostrada en todas las pestañas.</p>
<p><strong>PR (Performance Rating)</strong></p>
<blockquote>
<p>El error medio de equidad por decisión contada, multiplicado por 500 como hacen eXtreme Gammon y GNUbg: un PR de 5,0 vale 0,010 de equidad perdida por decisión, es decir 10 milipuntos (mpt). La regla de conteo exacta — qué decisiones entran en el denominador, cómo se convierte el marcador — es la de Anexo: Modelo de estadísticas — alineación XG / gnuBG / blunderDB.</p>
<p>Las bandas de nivel que el panel dibuja detrás de la curva de progresión son una <strong>referencia indicativa propia de blunderDB</strong>: ninguna publicación es autoridad sobre estos umbrales. El límite superior de cada banda queda excluido: un PR de 4 es <em>Avanzado</em>, no <em>Experto</em>.</p>
<table>
<thead>
<tr>
<th>Nivel</th>
<th>PR</th>
</tr>
</thead>
<tbody>
<tr>
<td>Clase mundial</td>
<td>&lt; 2</td>
</tr>
<tr>
<td>Experto</td>
<td>2 – 4</td>
</tr>
<tr>
<td>Avanzado</td>
<td>4 – 6</td>
</tr>
<tr>
<td>Intermedio</td>
<td>6 – 9</td>
</tr>
<tr>
<td>Ocasional</td>
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
<p>Probabilidad acumulada de victoria del match perdida a causa de los errores, sobre el conjunto de datos filtrado. Calculada a partir de la MET Kazaross-XG2 integrada en blunderDB.</p>
<div class="admonition caution">
<p>El MWC cost <strong>no es aplicable</strong> a las posiciones de <em>money-game</em> (sin apuesta de match). Esas posiciones se excluyen del cálculo de MWC. Los valores de MWC dependen de la MET utilizada; no son directamente comparables entre programas que usan METs diferentes.</p>
</div>
</blockquote>
<p>El cambio PR ↔ MWC es instantáneo: no se realiza ningún recálculo en el backend.</p>
<h4>El informe HTML</h4>
<p>El botón <strong>Informe HTML</strong> de la cabecera del panel produce un documento <strong>autónomo</strong>: un solo fichero, sin imagen externa, sin hoja de estilo remota, sin script. Los diagramas son SVG en línea, dibujados por el mismo renderizador que el tablero en pantalla, con su paleta. Se abre en cualquier navegador, viaja por correo electrónico, y <strong>se imprime en PDF desde el propio navegador</strong> — lo que evita incorporar un generador de PDF para producir lo que todo el mundo ya tiene.</p>
<p>Contiene los indicadores del ámbito actual (posiciones, partidos, decisiones contadas, PR global, de fichas y de cubo), luego las <strong>diez decisiones más costosas</strong>, cada una con su diagrama, su coste, el partido del que viene y la mejor jugada cuando un análisis la da.</p>
<p>El informe lleva el <strong>filtro actual</strong> del panel Estadísticas. Un informe que no dice su ámbito es un informe cuyas cifras no significan nada: ajuste el filtro — un torneo, un rango de fechas, un jugador — antes de producirlo.</p>
<h4>Pestaña Dashboard</h4>
<p>La pestaña <strong>Dashboard</strong> ofrece una vista sintética de los indicadores clave.</p>
<h5>Tarjetas de nivel</h5>
<p>Tres tarjetas muestran el PR (o MWC) para:</p>
<ul>
<li><strong>PR Global</strong> — todas las decisiones (fichas + cubo);</li>
<li><strong>PR Ficha</strong> — solo jugadas de fichas;</li>
<li><strong>PR Cubo</strong> — solo decisiones de cubo.</li>
</ul>
<p>Hacer clic en una tarjeta carga en el panel de análisis las posiciones del subconjunto correspondiente (drill-down).</p>
<div class="admonition note">
<p>El número total de decisiones se muestra en la parte inferior de cada tarjeta al pasar el cursor.</p>
</div>
<h5>PR deslizante sobre las últimas N decisiones</h5>
<p>Una fila de valores PR (o MWC) calculados sobre las últimas <em>N</em> decisiones (N = 5, 10, 50, 100, 250, 500, 1000) permite medir la tendencia reciente. Los valores atenuados corresponden a un N superior al número de decisiones disponibles.</p>
<p>Hacer clic en un valor carga las últimas <em>N</em> posiciones correspondientes.</p>
<h5>Top blunders</h5>
<p>La lista de los 10 peores errores (o MWC cost), ordenados por magnitud decreciente. Hacer clic en una fila carga la posición implicada en el panel de análisis.</p>
<h4>Pestaña Progresión</h4>
<p>La pestaña <strong>Progresión</strong> presenta la evolución del nivel a lo largo del tiempo.</p>
<p>En la cabecera de la pestaña, un <strong>objetivo</strong>: «PR &lt; 5 en doce semanas». Una meta, un plazo, y una tendencia que dice hacia dónde se va — nada más. Un objetivo que se pusiera a calificar, felicitar o recordar sería otra función, no esta.</p>
<p>El botón <strong>Proponer</strong> sugiere una meta a partir del nivel actual: el límite inferior de la banda en la que está, es decir, la entrada en la siguiente. Proponer «un poco mejor» no se anclaría en nada; proponer un escalón dice algo — pasar de intermedio a avanzado se ve y se cuenta.</p>
<p>La <strong>tendencia</strong> es un ajuste por mínimos cuadrados sobre el PR de sus partidos, proyectado al plazo. Se niega a pronunciarse por debajo de tres partidos: trazar una recta entre dos puntos sería una afirmación que no se puede sostener. Y la frase lo dice cada vez — <em>una tendencia no es una predicción</em>.</p>
<p>El objetivo se guarda en los <strong>metadatos de la base</strong>, no en la configuración: se refiere a esa biblioteca, así que sigue al fichero y no a la máquina. Ningún cambio de esquema: <code>metadata</code> ya es una tabla de claves y valores, legible por <code>blunderdb info</code> como por el demonio.</p>
<h5>Curva por torneo</h5>
<p>Un gráfico de líneas muestra el PR (o MWC) para cada torneo (eje X: orden de los torneos, eje Y: valor de la métrica). Unas bandas de color materializan los umbrales de nivel.</p>
<p>Hacer clic en un punto del gráfico abre un menú contextual con dos opciones:</p>
<ul>
<li><strong>Abrir torneo</strong> — abre el torneo en el panel Torneos.</li>
<li><strong>Abrir posiciones</strong> — carga todas las posiciones del torneo en el panel de análisis.</li>
</ul>
<h5>Gráfico de dispersión por partida</h5>
<p>Un diagrama de dispersión representa cada partida (eje X: fecha, eje Y: PR o MWC). El tamaño del punto es proporcional al número de decisiones de la partida.</p>
<p>Hacer clic en un punto abre un menú contextual:</p>
<ul>
<li><strong>Abrir partida</strong> — abre la partida en el panel de partidas.</li>
<li><strong>Abrir posiciones</strong> — carga todas las posiciones de la partida en el panel de análisis.</li>
</ul>
<h4>Pestaña Errores</h4>
<p>La pestaña <strong>Errores</strong> desglosa las fuentes de error.</p>
<h5>Reparto por acción de cubo</h5>
<p>Un diagrama de barras muestra el PR (o MWC) para cada tipo de decisión de cubo: <em>NoDouble</em>, <em>DoubleTake</em>, <em>DoublePass</em>, <em>TooGood</em>. Cada barra indica también el número de decisiones y la tasa de blunders en una información emergente.</p>
<p>Hacer clic en una barra carga las posiciones correspondientes a esa acción de cubo, <strong>solo las que tienen un error</strong> (drill-down).</p>
<h5>Dirección de los errores de cubo</h5>
<p>El reparto anterior indica <em>cuánto</em> cuestan las decisiones de cubo; esta tabla indica en <em>qué sentido</em> se equivocan.</p>
<p>Una posición de cubo lleva dos decisiones tomadas por dos jugadores distintos, presentadas aquí en dos filas:</p>
<ul>
<li><strong>Ofrecer</strong> — el jugador que tiene el cubo dobla o no dobla. Sus errores son los <strong>dobles perdidos</strong> (había que doblar) y los <strong>dobles prematuros</strong> (no había que hacerlo).</li>
<li><strong>Responder</strong> — el jugador al que se ofrece el cubo toma o pasa. Sus errores son los <strong>pases erróneos</strong> (se pasó una toma correcta) y las <strong>tomas erróneas</strong> (se tomó un pase correcto).</li>
</ul>
<p>Las dos filas se mantienen separadas a propósito: un jugador puede perfectamente doblar tarde <em>y</em> tomar holgado, y un indicador único llamaría a eso «equilibrado» perdiendo las dos mitades de la información.</p>
<p>Cada casilla muestra el número de decisiones; la información emergente da la equidad perdida acumulada. Hacer clic en una casilla carga las posiciones correspondientes. Una casilla a cero no es pulsable.</p>
<div class="admonition note">
<p>Esta tabla cuenta decisiones, no emite juicios. A partir de qué diferencia una tendencia merece ser nombrada depende del tamaño de la muestra y de un punto de referencia, que no son datos del motor.</p>
</div>
<h5>Comparación Checker / Cube</h5>
<p>Un diagrama comparativo coloca lado a lado el PR de las jugadas de fichas y de las decisiones de cubo. Hacer clic en una barra carga las posiciones del subconjunto con error.</p>
<h5>Histograma de las magnitudes de error</h5>
<p>Un histograma distribuye los errores según su magnitud en milésimas de punto (intervalos: 0–5, 5–10, 10–25, 25–50, 50–100, ≥ 100). Hacer clic en una barra carga las posiciones del intervalo.</p>
<h4>Pestaña Desgloses</h4>
<p>La pestaña <strong>Desgloses</strong> divide las mismas decisiones que cuentan las cifras globales según cuatro ejes. Ninguno redefine qué cuenta como decisión: sería un segundo PR con el mismo nombre.</p>
<ul>
<li><strong>Por fase de la partida</strong> — apertura, medio juego, carrera, retirada de fichas. Es lo que responde a «mi PR en carrera frente a mi PR en contacto». La etiqueta se calcula a partir del tablero (véase Panel de Búsqueda); una base cuyas fases nunca se han calculado lo ordena todo bajo <em>Sin clasificar</em>, y <code>blunderdb repair</code> la rellena.</li>
<li><strong>Por plan de juego</strong> — carrera, blitz, ancla, backgame, bloqueo contra bloqueo… Es el desglose para el que existe el clasificador: «¿dónde pierdo más?», plan por plan. La misma etiqueta derivada que la fase, las mismas reservas, y <code>blunderdb repair</code> la rellena igual.</li>
<li><strong>Por etiqueta</strong> — las <code>#palabra</code> escritas en los comentarios. Una posición puede llevar varias: <strong>estas filas no suman el total</strong>, y el panel lo dice bajo la tabla. Una etiqueta califica, no particiona.</li>
<li><strong>Por marcador</strong> — los puntos que faltan a ambos bandos, leídos del lado del jugador en turno, es decir del lado de quien decide. La fila <em>Money</em> es la partida por dinero. Una celda con menos de diez decisiones aparece <strong>en gris con su efectivo visible</strong> en lugar de oculta: demasiado poco para leerse, pero la omisión sigue siendo verificable.</li>
</ul>
<div class="admonition note">
<p>La partida Crawford no se distingue: blunderDB no registra ese indicador en una posición. El efecto práctico es pequeño — una partida Crawford no tiene ninguna decisión de cubo — pero la omisión es real y vale más escribirla que dejarla adivinar.</p>
</div>
<h4>Estudio y juego real</h4>
<p>El comando <code>blunderdb list --type study --days 30</code> pone tres números uno al lado del otro, plan por plan: cuántas <strong>posiciones distintas</strong> se repasaron en el periodo, cuál era el PR <strong>antes</strong>, cuál es el PR <strong>desde entonces</strong>.</p>
<p>Tres números, y ningún cuarto. <strong>No hay columna de ganancia ni flecha</strong>, porque aquí nada controla nada: el jugador pudo encontrarse con rivales más fuertes, cambiar de formato, o simplemente jugar más carreras este mes. La comparación es del lector; una columna que anunciara un efecto afirmaría una causalidad que estos datos no sostienen. Los números, en cambio, son exactos.</p>
<p>Los repasos se cuentan en <strong>posiciones distintas</strong>: una tarjeta repasada cuatro veces en el mes es una posición estudiada, y contar las repeticiones haría que un mes de empolle pareciera un mes de cobertura. Las decisiones del PR, en cambio, se cuentan todas — cada una se tomó una vez. Un PR apoyado en menos de diez decisiones se muestra <code>—</code>, con su muestra visible al lado.</p>
<h4>Pestaña Jugadores</h4>
<p>Las cuatro pestañas anteriores describen a <strong>un</strong> jugador; la pestaña <strong>Jugadores</strong> los compara a todos. Muestra una fila por jugador de la base, lo que responde a la necesidad de un organizador que sigue una competición entera más que a un jugador.</p>
<p>Columnas, por orden:</p>
<table>
<thead>
<tr>
<th>Columna</th>
<th>Significado</th>
</tr>
</thead>
<tbody>
<tr>
<td>Jugador</td>
<td>El nombre <strong>tal como figura en las partidas</strong>. Un jugador registrado con dos grafías aparece, pues, en dos filas; use la fusión de jugadores para reunirlas.</td>
</tr>
<tr>
<td>Partidas</td>
<td>Número de partidas disputadas en el periodo retenido.</td>
</tr>
<tr>
<td>V–D</td>
<td>Victorias y derrotas. Una partida inacabada (registro truncado, abandono) no cuenta ni la una ni la otra: V + D puede ser, pues, inferior al número de partidas.</td>
</tr>
<tr>
<td>Decisiones</td>
<td>Número de decisiones contadas — el denominador del PR. Es la columna que dice cuánto valen las tasas vecinas: un PR calculado sobre doce decisiones no significa nada.</td>
</tr>
<tr>
<td>PR</td>
<td>Performance Rating global.</td>
</tr>
<tr>
<td>PR fichas, PR cubo</td>
<td>El PR desglosado por tipo de decisión.</td>
</tr>
<tr>
<td>Snowie</td>
<td>Snowie Error Rate (véase Anexo: Modelo de estadísticas — alineación XG / gnuBG / blunderDB).</td>
</tr>
<tr>
<td>Blunders</td>
<td>Número de errores graves (al menos 0,100 EMG).</td>
</tr>
<tr>
<td>Suerte</td>
<td>Suerte media por tirada, en milésimas de punto, con signo: positiva si los dados fueron favorables.</td>
</tr>
</tbody>
</table>
<p>Uso:</p>
<ul>
<li><strong>Ordenar</strong> — haga clic en un encabezado de columna. La tabla se abre ordenada por PR creciente, mejor jugador primero. Los jugadores de los que nada se ha medido permanecen abajo sea cual sea el sentido de la ordenación: un cero por falta de datos no es una actuación perfecta.</li>
<li><strong>Abrir el detalle de un jugador</strong> — haga clic en una fila. El jugador queda seleccionado en la barra de filtros y la vista cambia a la pestaña Dashboard.</li>
<li><strong>Restringir el periodo</strong> — los filtros de fechas, de torneos y de longitud de partida se aplican con normalidad, lo que permite acotar la tabla a las fechas de una competición.</li>
</ul>
<div class="admonition note">
<p>En esta pestaña, la lista <strong>Jugador</strong> y la elección del <strong>tipo de decisión</strong> están desactivadas: la tabla muestra a todos los jugadores y ya desglosa las decisiones de fichas y de cubo en columnas distintas.</p>
</div>
<div class="admonition important">
<p>Un guion («—») señala un valor <strong>nunca medido</strong>, que no debe confundirse con cero. Es en particular el caso de la columna Suerte para toda partida importada antes de la versión 2.15.0 del esquema: la suerte no se conservaba entonces, y nada permite reconstruirla después — hay que reimportar los archivos de origen. Los formatos que no la transportan (BGF, Jellyfish <code>.mat</code>) no la aportarán nunca.</p>
</div>
<h4>Regla de agregación</h4>
<div class="admonition important">
<p>El PR de un torneo (o de cualquier subconjunto) se calcula mediante la regla <strong>suma/suma</strong>, nunca como media de los PR individuales de las partidas.</p>
<p>Fórmula:</p>
<pre class="math">PR_&#123;torneo&#125; = 500 \\times \\frac&#123;\\sum_&#123;i&#125; \\text&#123;error&#125;_i&#125;&#123;\\text&#123;número total de decisiones&#125;&#125;</pre>
<p><strong>Ejemplo:</strong> un jugador disputa dos partidas en un torneo —</p>
<ul>
<li>Partida A: 10 decisiones, 0,100 de equidad perdida → PR = 5,0</li>
<li>Partida B: 90 decisiones, 0,540 de equidad perdida → PR = 3,0</li>
</ul>
<p>Media ingenua de los PR: (5,0 + 3,0) / 2 = <strong>4,0</strong> <em>(incorrecto)</em></p>
<p>Regla suma/suma: 500 × 0,640 / (10 + 90) = <strong>3,2</strong> <em>(correcto)</em></p>
<p>La regla suma/suma es la única que maneja correctamente la variación de longitud de los matches (un match a 21 puntos pesa más que un match a 1 punto).</p>
</div>
<h4>MWC: limitaciones</h4>
<ul>
<li>El MWC cost se calcula a partir de la <strong>MET Kazaross-XG2</strong>, tabla de referencia de facto en el backgammon competitivo. Los resultados no son directamente comparables con programas que usan otras MET. Es la misma tabla, leída por el mismo punto de entrada, que la que usa el evaluador integrado para sus decisiones de cubo a un marcador de match: las estadísticas y el motor no pueden divergir en esto. Da sus propios valores hasta 25 puntos por hacer de cada lado; más allá, se prolonga con una tabla de Zadeh calculada como la de GNUbg, hasta 64.</li>
<li>Las posiciones de <em>money-game</em> (sin puntuación de match) se <strong>excluyen</strong> del cálculo de MWC. Si su base de datos contiene muchas posiciones de money-game, el MWC cost puede estar subestimado o no estar disponible.</li>
<li>El MWC cost es acumulativo sobre el conjunto de datos filtrado, no es un indicador por decisión. Mide el impacto total de sus errores sobre sus posibilidades de victoria.</li>
</ul>
<h3>Panel Eval</h3>
<p>El panel <strong>Eval</strong> (<em>CTRL-E</em>) evalúa en directo cualquier posición que esté en el tablero; en una posición de bearoff se especializa y calcula además el EPC (Effective Pip Count). Se activa pulsando <em>CTRL-E</em>, haciendo clic en la pestaña Eval del panel inferior o ejecutando el comando <code>epc</code>. Este comando conserva su nombre de origen: el panel se llamó <em>EPC</em>, luego <em>Bearoff</em>, antes de convertirse en <em>Eval</em> — es por tanto aquí donde hay que buscar lo que una versión anterior llamaba el panel Bearoff, nombre que ya solo designa la pestaña de configuración de las tablas de bearoff.</p>
<p>El panel muestra siempre la <strong>única decisión</strong> que pide la posición colocada en el tablero — nunca dos a la vez — y los hechos que la acompañan. Cada magnitud se lee en el eje que le conviene y no en un eje único impuesto: la probabilidad de victoria, de gammon, de backgammon y la equidad cubeless de cada jugador, calculadas <em>antes de la tirada</em>, se leen <strong>por jugador</strong> (abajo, arriba, luego Δ), a la izquierda de la decisión de cubo, cuando no hay dados colocados. Los hechos y la decisión permanecen uno junto al otro: la decisión de cubo nunca pasa por debajo de las cifras que la justifican, sean cuales sean el idioma de la interfaz y la posición en el tablero. En cuanto hay dados colocados, esos mismos valores <em>antes de la tirada</em> cambian de eje: se leen <strong>al tiro</strong>, en cabeza de la lista de jugadas candidatas, en forma de una fila en cursiva <em>antes de la tirada</em> — no una jugada candidata más, sino una referencia contra la que leer cada jugada. La diferencia entre esa fila y una jugada contiene la suerte de la tirada, nunca el mérito de la jugada, y por eso no lleva ninguna columna de error. En una posición de bearoff puro, una segunda tabla, siempre <strong>por jugador</strong> y siempre presente, con o sin dados colocados, lleva el EPC, el pip count, el wastage, el número medio de tiradas y la desviación típica; esas cinco columnas nunca migran. Las dos tablas están apiladas y comparten la misma rejilla de columnas: mismos bordes, mismas referencias de columna, una sola columna de puntos de color — se leen como un solo objeto de dos pisos. El distintivo de régimen, la atribución del motor (la profundidad de la última evaluación figura también en ella) y la casilla <em>Desafío</em> forman una banda aparte, alineada a la derecha por encima de las tablas.</p>
<p>Solo la lista de jugadas candidatas se desplaza — la fila <em>antes de la tirada</em>, también ella, permanece fijada encima; el resto del panel (hechos, distintivo, decisión de cubo) permanece siempre visible, sin ningún ajuste particular del tamaño del panel.</p>
<p>La tabla de hechos y la decisión las calcula gammonNet, integrado, sin XG ni gnubg. El cálculo sigue la posición sin bloquear nunca la interfaz: una profundidad 0-ply se muestra de inmediato con cada gesto y luego, tras medio segundo de inmovilidad, una evaluación más profunda (2 plies por defecto, ajustable en la pestaña <em>gammonNet</em> de la configuración) la sustituye en segundo plano — cualquier nuevo gesto cancela ese cálculo de fondo. La profundidad mostrada en la banda de distintivos, o dentro del distintivo de régimen en una posición de carrera, es siempre la que ha producido efectivamente la cifra mostrada, nunca la solicitada; no se repite en cada fila, puesto que una evaluación en directo comparte la misma profundidad para todas las jugadas. La equidad de las jugadas candidatas y de la decisión de cubo sigue el marcador de la posición: en money game se expresa en puntos, en un marcador de match en <strong>equidad normalizada</strong> — la misma escala que XG y GNU Backgammon, donde ganar el valor del cubo actual vale +1 y perderlo −1 — nunca mezcladas en una misma tabla. El encabezado de la columna lo indica explícitamente en lugar de dejar la escala a adivinar: «Equity (money)» en money game, «Equity (match)» en un marcador de match. Tiene en cuenta el <strong>cubo vivo</strong>: la búsqueda valora cada posición final mediante el modelo de cubo (Janowski, eficiencia medida) en el estado del cubo de la posición, tal como hacen XG y GNU Backgammon en la evaluación <em>cubeful</em>. Esto es lo que hace visibles, al marcador, los efectos gammon-go y gammon-save — a 4-away/2-away, el jugador que va detrás juega 8/2 6/2 con una apertura de 6-4 porque su doblaje temprano dará al gammon el valor del match, algo que una evaluación sin cubo no puede ver. La fila <em>antes de la tirada</em>, por su parte, sigue siendo una equidad <strong>cubeless</strong>: es un hecho de la posición, no una decisión. Este panel nunca modifica la base: es un cálculo, no un análisis registrado. Hacer clic en una jugada candidata la muestra en el tablero en forma de flechas, exactamente como en el panel Análisis. El discreto botón <strong>?</strong>, en la banda de distintivos, lleva al repositorio del motor <code>gammonNet &lt;https://github.com/kevung/gammonNet&gt;</code>_; la atribución completa (red Strehl, configuración gammonNet) figura en los Agradecimientos de la ayuda.</p>
<p>El usuario edita la posición de las fichas en todo el tablero, exactamente como en modo edición: el clic izquierdo coloca una ficha del jugador de abajo, el clic derecho una ficha del jugador de arriba. La segunda tabla, la de la carrera, solo aparece cuando la posición obtenida es un bearoff puro (todas las fichas de ambos jugadores en su cuadrante); en cualquier otra posición, solo responde la tabla de las cuatro columnas comunes (victoria, gammon, backgammon, cubeless), y la decisión se refiere a las fichas o a un cubo genérico según haya o no dados colocados.</p>
<p>En cada tabla de hechos, una fila por jugador — identificada por su punto de color, con el jugador negro siempre abajo. La primera lleva, mientras no haya dados colocados, la victoria, el gammon, el backgammon (probabilidades, sin el signo %) y la equidad cubeless del jugador; la segunda, en una posición de bearoff y con o sin dados colocados, el EPC, el pip count, el wastage (diferencia entre el EPC y el pip count), el número medio de tiradas y la desviación típica. Cuando ambos jugadores tienen valores que comparar, una fila <strong>Δ</strong> da las diferencias <em>con signo</em> (abajo − arriba: negativa cuando el jugador negro va en cabeza). Fuera de una posición de carrera, colocar dados hace por tanto desaparecer las propias tablas de hechos: las cuatro columnas que llevaban acaban de cambiar de eje, al tiro, en cabeza de la lista de jugadas.</p>
<p>La decisión de cubo tiene siempre la misma forma, sea cual sea el origen de las cifras — tabla exacta, régimen evaluado o evaluación gammonNet ordinaria: <strong>una fila por opción</strong>, en el orden <em>no doblar</em>, <em>doblar/tomar</em>, <em>doblar/pasar</em>, con su equidad en el referencial de la posición y su diferencia respecto a la mejor opción. El orden nunca cambia, al contrario que la lista de jugadas: las tres opciones tienen nombre, así que se lee el nombre, no el rango. La mejor se reconoce por su realce y por su celda de diferencia dejada vacía. Cuando el cubo ya se ha girado, las opciones se leen <em>no redoblar</em>, <em>redoblar/tomar</em>, <em>redoblar/pasar</em>.</p>
<p>Una última fila da el <strong>veredicto</strong>. Toma cuatro valores: <em>no doblar</em>, <em>doblar, tomar</em>, <em>doblar, pasar</em> y <em>demasiado bueno para doblar</em>, este último cuando jugar la posición rinde más que cobrar el punto: doblar sería entonces un error por la razón inversa a la del simple <em>no doblar</em>. Es también el único lugar donde el panel dice que <strong>no</strong> hay veredicto, en lugar de dejar creer que hay un cálculo en curso:</p>
<ul>
<li><em>sin decisión</em> — el régimen no tiene derecho a ella; el veredicto de cubo nunca se estima (véase el distintivo <em>estimado</em>);</li>
<li><em>no evaluable en este marcador</em> — el motor rechaza la posición, típicamente un marcador fuera del horizonte de la tabla de equidad de match, es decir un bando a más de 64 puntos por hacer;</li>
<li><em>cubo adversario</em> y <em>cubo muerto (Crawford)</em> — el cubo no puede girarse. Las equidades siguen mostrándose, a título indicativo, pero ninguna opción lleva diferencia: un error es lo que cuesta una elección, y no hay elección.</li>
</ul>
<p>En money game, las reglas <strong>Jacoby</strong> y <strong>Beaver</strong> activas en la posición aparecen bajo la tabla del cubo, en pequeñas insignias junto al veredicto que modifican: el veredicto «no doblar» de una posición bajo la regla Jacoby no es el mismo cálculo que sin ella, y nada más en pantalla lo indicaba.</p>
<p>Una tercera insignia, <strong>Cubo máx.</strong>, aparece cuando el identificador de origen limita el cubo — tanto a un marcador de partido como en money game. Esa no describe el cálculo mostrado encima: el evaluador integrado no modela un techo, así que el veredicto es el de un cubo libre. Precisamente por eso está la insignia: un cubo limitado es la única razón visible por la que blunderDB y eXtreme Gammon pueden anunciar dos veredictos distintos sobre la misma posición.</p>
<p>El distintivo de régimen, la profundidad de evaluación, el enlace al motor y la casilla <em>Desafío</em> forman una banda aparte, alineada a la derecha por encima de las tablas.</p>
<p>El <strong>jugador al tiro</strong> y la <strong>posición del cubo</strong> se editan directamente en el tablero, como en modo edición: hacer clic en el rectángulo bearoff/marcador de un jugador le da el tiro; hacer clic en el cubo lo hace rotar centrado → poseído abajo → poseído arriba (clic derecho en sentido inverso). El valor del cubo permanece fijado — en money game las equidades se expresan en unidades del cubo actual, solo cuenta su propietario. El análisis se recalcula de inmediato. En régimen estimado, el propio distintivo es clicable y abre directamente la pestaña <em>Bearoff</em> de la configuración; su información emergente explica por qué (veredicto de cubo no estimable, <code>ADR-0009 &lt;https://github.com/kevung/blunderDB/blob/main/docs/adr/0009-race-win-chances-are-read-or-convolved-cube-verdicts-are-never-estimated.md&gt;</code>__) y cómo ampliar el dominio exacto.</p>
<p>El <strong>marcador</strong> se edita también directamente en el tablero, como en modo edición: el clic izquierdo en el rectángulo del marcador de un jugador decrementa su número de puntos por hacer, el clic derecho lo incrementa. Salir del marcador <em>money</em> (-1, -1) editando un solo bando alinea automáticamente el otro bando con el mismo valor en lugar de dejar un marcador incoherente. En una posición de bearoff en régimen <em>exacto</em>, pasar de un marcador money a un marcador de match deja la probabilidad de victoria tal cual (una lectura en base, válida sea cual sea el referencial) pero conmuta la equidad y el veredicto de cubo mostrados a los del régimen <em>evaluado</em> — la tabla exacta, al ser money por construcción, no sabe responder a la pregunta planteada en el marcador. El distintivo pasa entonces a ser compuesto (« exacto (victoria) · evaluado (cubo) ») para decirlo explícitamente.</p>
<p>Los <strong>dados</strong>, por último, se editan de la misma manera, y son ellos los que deciden la pregunta planteada: con dados colocados se trata de una decisión de fichas (la lista de jugadas candidatas), sin dados de una decisión de cubo. Un clic izquierdo en un dado sube su valor (el 6 vuelve al 1), un clic derecho lo baja (el 1 vuelve al 6); hacer clic en un dado en un tablero que no tiene coloca dos de golpe — un solo dado no sería ni una decisión de fichas ni una decisión de cubo. Hacer clic en el rectángulo de un jugador retira los dados para plantear una pregunta de cubo, y el siguiente clic en un dado los vuelve a poner tal como estaban.</p>
<p><em>RETROCESO</em>, o un doble clic fuera del tablero, borra la posición: tablero vacío, marcador money (-1, -1), sin dados colocados — valores propios del panel Eval, distintos de los usados en modo edición (7 en ambos lados, dados 3-1), para mantener la coherencia con lo que el panel muestra por defecto.</p>
<h4>Matriz del cubo</h4>
<p>Una decisión de cubo no es una propiedad del tablero. Las mismas fichas, el mismo recuento de pips, se doblan a 2-away/4-away y no se doblan a 4-away/2-away; quien ha aprendido la respuesta money ha aprendido una casilla de una cuadrícula. El panel Eval muestra la casilla que la posición lleva; la <strong>matriz del cubo</strong> muestra la cuadrícula entera.</p>
<p>El comando <code>cm</code> la abre sobre la posición mostrada. Cada casilla da el veredicto en un marcador: la fila es el número de puntos que aún necesita el jugador en turno, la columna el del adversario. Los cuatro veredictos se escriben <em>ND</em> (no doblar), <em>DT</em> (doblar, tomar), <em>DP</em> (doblar, pasar) y <em>DB</em> (demasiado bueno); una casilla que el motor rechaza lleva un signo de interrogación y dice por qué al pasar el ratón, que da además las tres equidades de la casilla. Se ofrecen tres longitudes de partido: 5, 7 y 9 puntos.</p>
<p>El marcador de la posición se sustituye por el de cada casilla; su <strong>cubo</strong> se conserva. La cuadrícula responde a a qué marcador giraría <em>este</em> cubo, no a lo que haría una posición centrada. Es posterior a Crawford de principio a fin: durante la partida Crawford el cubo no está en juego, y una columna de «no puede doblar» no diría nada de la posición.</p>
<p>Cada casilla es una búsqueda propia. El motor tiene en cuenta el marcador — no juega la misma partida a 2-away que a 7-away —, así que una sola búsqueda releída a través de equidades de partido distintas sería falsa justo donde el marcador importa. La cuadrícula llega primero en 0-ply y se recalcula a la profundidad de visualización configurada cuando la ventana queda en reposo: la misma escalada que el resto del panel, para una cuadrícula de 9 puntos que cuesta alrededor de un segundo y medio.</p>
<p>La misma cuadrícula se calcula fuera de la interfaz, con el comando cubematrix de la línea de comandos.</p>
<h4>Llevar una posición al panel Eval</h4>
<p>El panel se abre por defecto en una posición de bearoff, pero el estudio parte la mayoría de las veces de una posición que ya se tiene a mano. Dos gestos la llevan allí:</p>
<ul>
<li><strong>Clic derecho en el tablero</strong>, en un panel de análisis o durante la navegación de una partida, y luego <em>Evaluar esta posición</em>: el panel Eval se abre directamente en esa posición, tal como se muestra. El menú contextual no aparece en el panel Eval ni en el panel de búsqueda, donde el botón derecho ya sirve para colocar las fichas del otro color.</li>
<li><strong>CTRL-C y luego CTRL-V</strong>: copiar la posición desde el panel de análisis y pegarla después, una vez, en el panel Eval. El pegado acepta también un identificador venido de otra parte — un XGID (eXtreme Gammon, GNU Backgammon, otra instancia de blunderDB) o un OGID (OpenGammon): basta con que esté en el portapapeles.</li>
<li><strong>El comando</strong> <code>import XGID=…</code> (o <code>import OGID=…</code>) para cuando el identificador no está en el portapapeles sino en un mensaje, en un foro leído en un terminal, o producido por un script. Es el mismo verbo que <code>import</code> a secas: sin argumento abre un selector de ficheros, con argumento lee el identificador. El camino es luego idéntico al del pegado — misma lectura, misma deduplicación, misma apertura de la posición importada.</li>
</ul>
<p>Un OGID solo lleva una posición: ni evaluación, ni comentario. La posición llega por tanto sin análisis, exactamente como un XGID desnudo, y el evaluador integrado puede rellenar el hueco después.</p>
<p>El tablero del panel Eval es un borrador: la posición llega a él sin su identificador de base, de modo que ninguna modificación hecha aquí puede reescribir el registro del que procede. Todas las ediciones habituales del tablero siguen disponibles en él (fichas, cubo, dados, marcador), y la evaluación sigue cada modificación.</p>
<p>En sentido inverso, <em>CTRL-C</em> copia el tablero del panel Eval al portapapeles, con un XGID recalculado a partir de las fichas colocadas — y por tanto pegable directamente en eXtreme Gammon o en otra instancia de blunderDB. Solo viaja la posición: la evaluación mostrada por el panel no es un registro de la base y no acompaña a la copia.</p>
<p>Al salir del panel Eval, se restaura la posición consultada anteriormente: el borrador nunca se guarda por sí solo.</p>
<p>Cuando la posición es un bearoff puro (todas las fichas de ambos jugadores en su cuadrante) y no hay dados colocados, la decisión de cubo muestra, para el jugador al tiro:</p>
<ul>
<li>en régimen <em>exacto</em>: las equidades money (cubeless, sin doblar, doblar/tomar, doblar/pasar) y el <strong>veredicto de cubo money</strong> (no doblar, doblar/tomar, doblar/pasar o demasiado bueno para doblar) — fuera del marcador de match, véase más arriba para el caso del marcador,</li>
<li>en régimen <em>evaluado</em>: las mismas equidades y el mismo veredicto de cuatro valores, pero <strong>jugados por gammonNet</strong> (búsqueda + modelo de cubo de Janowski) en lugar de leídos en una tabla — disponibles <strong>incluso en un marcador de match</strong>, lo que el régimen estimado nunca pudo ofrecer;</li>
<li>en régimen <em>estimado</em>: el veredicto de cubo, deliberadamente, no se muestra entonces — solo la probabilidad de victoria, en la tabla de hechos, acompañada de su margen de error, sigue disponible.</li>
</ul>
<p>En cuanto hay dados colocados en una posición de carrera, esa decisión de cubo <em>antes de la tirada</em> desaparece — el tablero pide entonces una decisión de fichas, no de cubo — pero la probabilidad de victoria, por su parte, sigue siendo un hecho de la posición, no una decisión: se une a la fila <em>antes de la tirada</em> en cabeza de la lista de jugadas, junto al EPC, que permanece mostrado justo a la izquierda.</p>
<p>Un distintivo indica el régimen: <strong>exacto</strong> (valor leído en una base de datos two-sided), <strong>evaluado · &lt;profundidad&gt;</strong> (jugado por gammonNet — la profundidad mostrada es la que ha producido efectivamente la cifra mostrada), <strong>estimado ± margen</strong> o, en un marcador de match dentro del dominio exacto, <strong>exacto (victoria) · evaluado (cubo)</strong> — véase más arriba. El régimen exacto prevalece allí donde está disponible; si no, el régimen evaluado se muestra en cuanto termina de calcular, sustituyendo en el sitio al régimen estimado mostrado durante la espera. Véase Metodología e hipótesis del panel Eval para la definición precisa de los tres regímenes y de sus hipótesis.</p>
<p><strong>Ampliar el dominio exacto.</strong> La tabla calculada en el primer arranque cubre 6 fichas por bando. Dos maneras de ir más allá, en la pestaña <em>Bearoff</em> de la configuración:</p>
<ul>
<li>calcular una tabla de dos lados más amplia — hasta TS-06-15 si la máquina tiene memoria para ello. La pestaña indica el tamaño, la memoria y el tiempo en esta máquina antes de empezar, y el cálculo se pausa y se reanuda. Un cálculo cancelado deja un archivo <code>.part</code> que nunca se lee como una tabla;</li>
<li>indicar cualquier archivo <code>.bd</code> two-sided de gnubg. La base con el dominio más amplio prevalece automáticamente.</li>
</ul>
<p><strong>El tablero del panel es un borrador, y se recuerda.</strong> Salir del panel Eval y volver recupera la posición en la que se dejó, no el tablero de bearoff por defecto: este solo se sirve la primera vez que se abre el panel en una sesión. Enviar una posición de la base al panel prevalece sobre ese recuerdo, y <em>RETROCESO</em> devuelve el tablero por defecto en cualquier momento. Nada se guarda en la base por el camino: el borrador no tiene identidad de posición, y su evaluación se recalcula al llegar en lugar de transportarse.</p>
<p><strong>Modo desafío.</strong> La casilla <em>Desafío</em>, en la banda de distintivos, activa un modo de entrenamiento: con cada modificación de la posición, los valores de tres zonas se ocultan (sustituidos por « ··· »); un clic en una zona revela solo esa zona. Sin dados, son la fila del jugador de abajo, la fila del jugador de arriba y la decisión de cubo — la fila Δ solo aparece una vez reveladas las dos filas de jugadores. El bloque de decisión conserva entonces sus tres filas: lo que desaparece son sus valores, su veredicto y el realce de la mejor opción, sin lo cual el ejercicio se resolvería buscando la fila en negrita. Con dados colocados en una posición de carrera, la fila EPC de cada jugador se oculta como antes, pero la tercera zona cubre entonces la fila <em>antes de la tirada</em> y la lista de jugadas <strong>juntas</strong>: como la lista está ordenada de la mejor jugada a la peor, revelarla parcialmente ya daría la respuesta. Con dados colocados fuera de una posición de carrera, esa misma zona única cubre por sí sola todo lo que el panel muestra. Así se puede entrenar a estimar el EPC de cada bando, y luego a pronunciarse sobre el cubo o sobre la jugada a realizar, antes de comprobar. El ajuste se memoriza.</p>
<p>Para cerrar el panel Eval, pulse <em>CTRL-E</em> o cambie a otra pestaña.</p>
<h4>Metodología e hipótesis del panel Eval</h4>
<p>Cada valor mostrado por el panel se basa en hipótesis precisas, enunciadas aquí exhaustivamente.</p>
<p><strong>Dominio.</strong> La <em>zona de carrera</em> — probabilidad de victoria y veredicto de cubo — solo trata el bearoff puro: todas las fichas restantes de ambos jugadores en su cuadro interior. La posición se evalúa <em>antes de la tirada</em>; los dados puestos se ignoran.</p>
<p>Los <strong>bloques EPC</strong>, en cambio, van más lejos: un bando obtiene su EPC en cuanto su ficha más lejana cabe en la tabla de un lado cargada. Con la tabla predeterminada (seis puntos) es la antigua regla del cuadro; con una tabla de ocho puntos, calculada desde la pestaña <em>Bearoff</em>, un bando con una ficha en la 8 se trata como cualquier otro. Nada se extrapola: una ficha un punto demasiado lejos simplemente no tiene EPC, exactamente como una ficha en la 7 no lo tenía antes. Cuando la tabla que respondió no es la de seis puntos, su nombre aparece en la esquina del bloque de carrera («OS-08») — sin él se leería «seis» por defecto y se creería al bando enteramente en casa.</p>
<p><strong>Bloques EPC (siempre exactos).</strong> El EPC, el número medio de tiradas y la desviación típica provienen de la distribución exacta del número de tiradas para sacar todas las fichas, leída en la base de un lado de GNUbg (6 a 10 puntos, 15 fichas, calculada en la máquina). EPC = tiradas medias × 49/6 (49/6 ≈ 8,167 es la media exacta de pips por tirada, dobles contados cuatro veces); wastage = EPC − pip count. La única idealización es el <em>juego óptimo de un lado</em>: cada jugador minimiza sus propias tiradas ignorando al adversario — es la definición estándar del EPC.</p>
<p><strong>Probabilidad de victoria, régimen exacto.</strong> Lectura directa en la base two-sided disponible más amplia (TS-06-06 calculada en el primer arranque, archivo externo, o TS-06-11 calculada desde la pestaña <em>Bearoff</em>). Estas bases resultan de un análisis retrógrado completo bajo juego two-sided óptimo de ambos bandos: ninguna hipótesis adicional, error limitado a la cuantificación (&lt; 0,002 %).</p>
<p><strong>Probabilidad de victoria, régimen estimado.</strong> Fuera del dominio de la base: la probabilidad se obtiene convolucionando las dos distribuciones one-sided (el jugador al tiro gana si su número de tiradas es inferior o igual al del adversario) y aplicando luego una corrección polinómica fija, calibrada fuera de línea contra la base TS-06-11. Tres hipótesis:</p>
<ul>
<li><strong>independencia</strong> de los dos procesos de salida — estructural en carrera: sin contacto no hay ninguna interacción;</li>
<li><strong>juego one-sided óptimo de ambos bandos</strong> — esta es <em>la aproximación</em>: en realidad el jugador que va detrás se desvía para jugar la varianza y el líder por seguridad. El efecto medido es un sesgo antisimétrico (la convolución exagera la ventaja del líder) que la corrección absorbe estadísticamente;</li>
<li>la <strong>corrección</strong> fue calibrada y validada sobre el dominio del oráculo (hasta 11 fichas por jugador). Error residual medido: desviación típica 0,05 %, percentil 99 0,17 %, máximo observado 0,9 % (en puntos de probabilidad de victoria). <strong>Más allá de 11 fichas por jugador, esta cota está extrapolada</strong> — la tendencia es monótona pero ningún oráculo la certifica.</li>
</ul>
<p><strong>Equidades y veredicto de cubo (solo régimen exacto).</strong> Las equidades mostradas son las del <strong>money game, sin Jacoby</strong>, en el referencial de la literatura del bearoff. En el dominio ≤ 11 fichas por jugador, los gammons son imposibles (cada bando ya ha sacado al menos 4 fichas): no es una aproximación. El veredicto (no doblar / doblar, tomar / doblar, pasar) se reconstruye exactamente a partir de las equidades almacenadas, según la regla de GNUbg, validada punto por punto contra su análisis.</p>
<div class="admonition note">
<p>Las equidades cubeful suponen un <strong>juego de cubo óptimo de ambos bandos hasta el final</strong>: los futuros redobles se valoran íntegramente (análisis retrógrado completo). En las carreras muy volátiles de final de partida, la cascada de redobles se come casi toda la ventaja del bando al tiro — las equidades « sin doblar » y « doblar/tomar » pueden entonces estar próximas a cero allí donde un motor como XG, cuyo modelo de cubo no valora esa cascada, muestra valores próximos al dead cube (por ejemplo, 2 fichas en el punto 3 contra 2 fichas en el punto 2: 62 % de victoria, D/T exacto +0,006 frente a +0,475 en XG). La <strong>decisión</strong> mostrada, en cambio, coincide con la de los motores.</p>
</div>
<p><strong>Probabilidad de victoria y veredicto, régimen evaluado.</strong> Fuera del dominio exacto, la probabilidad de victoria procede de la salida bruta de gammonNet (búsqueda a 0 o 2 plies según el gesto, nunca leída en una tabla), y el veredicto de un « Decide » de Janowski aplicado a esa salida — la búsqueda <em>juega</em> la trayectoria en lugar de resumir una instantánea de ella, que es precisamente lo que el régimen estimado no podía hacer (véase más abajo) y permite, único de los tres regímenes junto con el exacto, un veredicto <strong>en un marcador de match</strong>.</p>
<p>Este régimen se ha medido, no solo supuesto, contra la tabla two-sided integrada (<code>TestEvalMeasure</code>, 4000 decisiones money muestreadas, parámetros canónicos 2 plies k=12): concordancia del veredicto money <strong>93,4 %</strong> (3735/4000), desglosada por distancia al punto de toma de gammonNet — 61,1 % a menos del 1 % del punto de toma (la zona más sensible a un cara o cruz), 88,3 % entre el 1 y el 5 %, 91,5 % entre el 5 y el 10 %, 94,0 % entre el 10 y el 20 %, 94,4 % más allá. Diferencia de probabilidad de victoria: media 0,85 %, mediana 0,44 %, percentil 95 3,21 %, máximo 8,30 %. Diferencia de equidad cubeful: media 0,039, mediana 0,018, percentil 95 0,151, máximo 0,406. La forma es la esperada: lo esencial del desacuerdo se concentra exactamente en el punto de toma, donde dos métodos legítimamente distintos divergen más en una decisión ajustada — no un error difuso que costaría equidad en todas partes.</p>
<p>Esta medida se refiere a decisiones <strong>money</strong>, en carrera. El veredicto a un marcador de match — que solo este régimen sabe dar — y las posiciones de contacto no tienen medida publicada: lo anterior no se traslada a esos casos.</p>
<p><strong>¿Por qué no más profundo que 2 plies?</strong> Porque la medida dice que no aporta nada. Una decisión de fichas cuesta 99 ms a 2 plies y 8,4 s a 3 plies en la misma máquina — <strong>ochenta y cinco veces más</strong>. Sobre cuarenta decisiones reales repetidas a ambas profundidades, la búsqueda más profunda cambió de opinión <strong>dos veces</strong>, y ambas el beneficio que se atribuía a sí misma valía como mucho 0,0005 de equidad normalizada: dos órdenes de magnitud por debajo de 0,020, el umbral a partir del cual eXtreme Gammon habla de error. Por decisión, todos los casos juntos, el beneficio es de 0,0000.</p>
<p>El ajuste no se ofrece, por tanto. No se trata de decir que 3 plies no valga nada en general, sino que sobre <em>esta</em> red, con el filtro canónico, no paga la espera de quien está delante de un panel. La medida es reproducible (<code>TestThreePlyMeasure</code>) y la conclusión se volverá a juzgar si la red cambia.</p>
<p><strong>¿Por qué no existe el veredicto estimado?</strong> Lo que sigue se refiere específicamente al método por <em>convolución</em> (régimen estimado), no al régimen evaluado descrito arriba: la equidad cubeful es un problema de <em>trayectoria</em> (cuándo doblar) que ningún resumen estadístico de la posición captura — el mejor modelo estático medido deja un error residual (desviación típica 0,016 de equidad, máximo 0,20) que basta para invertir todas las decisiones ajustadas. Del mismo modo, la conversión del veredicto al marcador del match mediante una tabla de equidades de match se ha medido insuficiente (12 % de desacuerdos con el análisis 2-ply de GNUbg, con auténticos blunders). Como un veredicto falso mostrado con aplomo es peor que ningún veredicto, la convolución nunca ha tenido derecho a mostrar veredicto — es una búsqueda que juega la trayectoria, no un resumen estadístico, lo que rellena ese hueco.</p>
<div class="admonition note">
<p>Las bases de bearoff son tablas matemáticas inmutables. blunderDB las calcula él mismo, de forma idéntica a la herramienta <code>makebearoff</code> de GNUbg — byte a byte — en la pestaña <em>Bearoff</em> de la configuración o con <code>blunderdb bearoff generate</code>.</p>
</div>
<h3>Panel Anki</h3>
<p>El panel <strong>Anki</strong> (<em>CTRL-K</em>) permite estudiar posiciones mediante repetición espaciada utilizando el algoritmo FSRS. El usuario puede crear mazos a partir de colecciones o de resultados de búsqueda.</p>
<p><strong>Creación de mazos:</strong> Haga clic en <em>New Deck</em> para crear un mazo a partir de una colección o de los resultados de búsqueda actuales. Los mazos basados en una búsqueda se sincronizan automáticamente al activar la pestaña Anki.</p>
<p><strong>Repaso:</strong> Seleccione un mazo y luego haga clic en <em>Study</em> (o haga doble clic en un mazo) para empezar a repasar las cartas pendientes. Cada carta muestra la posición correspondiente en el tablero. Evalúe su recuerdo con las teclas <em>1</em> (Repetir), <em>2</em> (Difícil), <em>3</em> (Bien) o <em>4</em> (Fácil). Pulse <em>Esc</em> para detenerse y volver a la lista de mazos.</p>
<p><strong>Las decisiones de cubo hacen dos tarjetas, encadenadas.</strong> Una decisión de cubo son dos preguntas — «¿doblar?», luego «¿aceptar?» — y blunderDB siempre las ha guardado como dos posiciones. Un mazo que selecciona solo una mitad recibe la otra: la decisión se completa, no se amplía. Y cuando ambas vencen, la segunda viene <strong>inmediatamente</strong> después de la primera.</p>
<p>Cada una conserva su propia nota y su propio calendario: no son dos tiempos de una tarjeta, son dos tarjetas. El encadenamiento no adelanta ningún vencimiento — ordena las tarjetas ya vencidas, nada más. Como nacen juntas, vencen juntas la primera vez, y ahí es donde sirve.</p>
<p><strong>Mostrar la respuesta:</strong> La carta plantea una pregunta — qué jugada jugar, o qué acción de cubo. Reflexione y luego pulse <em>ESPACIO</em> (o haga clic en la zona oculta) para desvelar la respuesta: el análisis registrado de la posición, tal como lo presenta la pestaña Análisis. Aparece bajo los botones de evaluación, que permanecen en su sitio y al alcance. Hacer clic en una jugada de la lista la muestra en el tablero.</p>
<p>Nada le obliga a desvelar la respuesta para evaluar: si está seguro, las teclas <em>1</em> a <em>4</em> siguen activas. La respuesta vuelve a ocultarse en la carta siguiente, pero no si simplemente cambia de pestaña — vaya a consultar el panel Eval o el comentario de la posición, le estará esperando a la vuelta.</p>
<p>Una posición sin análisis registrado lo indica directamente, sin zona oculta.</p>
<p><strong>Limitar la sesión.</strong> De forma predeterminada, una sesión de repaso recorre todas las cartas pendientes. Puede acotarla a un número de cartas, por mazo, en los Ajustes: marque <em>Limitar la sesión</em> e indique cuántas cartas debe servir una sesión. Cuando se alcanza el límite, la sesión se detiene y lo dice — el mensaje distingue «límite alcanzado, quedan tantas cartas pendientes» de una cola realmente agotada. Para seguir de todos modos, ahí está la práctica libre: sirve otras posiciones sin modificar nada del calendario.</p>
<p>Un límite de <strong>0</strong> no sirve ninguna carta: es un estado por derecho propio, útil para congelar un mazo mientras se prepara un torneo, y no es lo mismo que «sin límite». El botón <em>Study</em> queda entonces inactivo.</p>
<p>El límite se aplica a la <strong>sesión</strong>, no al día. Un mazo de blunderDB se construye sobre una colección o sobre una búsqueda: es un corpus finito, introducido en unas pocas sesiones, cuyo volumen diario ya está acotado por su tamaño. Un tope diario nunca llegaría a morder, o bien crearía un atraso en un mazo que cabía en una sola sesión.</p>
<p><strong>Práctica libre (cram):</strong> El botón <em>Cram</em>, junto a <em>Study</em>, inicia una sesión de práctica libre: se le muestran posiciones aleatorias del mazo sin tener en cuenta el calendario FSRS. Este modo <strong>nunca modifica el plan de repetición espaciada</strong> — ideal para calentar antes de un torneo o repasar intensamente un mazo temático sin alterar su orden. Una etiqueta <em>Cram</em> reemplaza el estado de la carta y un botón <em>Siguiente</em> (teclas <em>1</em> a <em>4</em>) recorre las posiciones. <em>Esc</em> vuelve a la lista sin guardar una sesión interrumpida.</p>
<p><strong>Apartar una tarjeta, sin calificarla.</strong> Durante un repaso, un clic derecho en la cabecera de la tarjeta ofrece tres gestos que la sacan de la sesión sin decir nada al planificador:</p>
<ul>
<li><strong>Suspender</strong> — la tarjeta conserva su programación y no vuelve a salir mientras esté suspendida. Es la manera de apartar una tarjeta equivocada, o aún no útil, sin perder el historial asociado.</li>
<li><strong>Posponer</strong> — la tarjeta desaparece hasta el día siguiente. A diferencia de suspender, esto no dice nada de su valor: es para la que se acaba de ver en otro sitio, o que se prefiere no cruzar dos veces en una tarde.</li>
<li><strong>Quitar</strong> — la tarjeta abandona el mazo, tras confirmación. La posición permanece en la base: un mazo es una lista de estudio sobre la biblioteca, nunca una copia de ella.</li>
</ul>
<p>Ninguno de estos tres gestos registra una nota: una tarjeta apartada no es una tarjeta respondida, y no cuenta en el total de la sesión.</p>
<p><strong>Registro de repasos.</strong> En los Ajustes de un mazo, el botón <em>Registro de repasos</em> muestra lo que se le <strong>dijo</strong> al planificador — fecha, posición, nota, estado, intervalo concedido — frente a lo que planea. Es el único sitio donde se ve una nota introducida por error. Allí no se corrige: la programación queda fuera de alcance, y esa regla es justamente lo que hace útil el registro — el pasado no se reescribe, pero se puede conocer.</p>
<p><strong>Pausa/Reanudación:</strong> Puede interrumpir una sesión de repaso en cualquier momento con <em>Esc</em>. El botón cambia a <em>Resume</em> y muestra su progreso. Haga clic en él para retomar donde lo dejó.</p>
<p><strong>Gestión de mazos:</strong> Use los botones de acción para renombrar, sincronizar, reiniciar o eliminar mazos (se pide confirmación para estas dos últimas acciones). Los parámetros FSRS (retención objetivo, intervalo máximo, aleatoriedad) pueden configurarse por mazo en los Ajustes (icono de engranaje).</p>
<p><strong>Retención: el objetivo y la medida.</strong> La <em>retención objetivo</em> es su elección sobre el compromiso entre carga de trabajo y calidad del recuerdo: cuanto más alta, más se acortan los intervalos y más repasa. Frente a ella, los Ajustes muestran la <strong>retención medida</strong> sobre sus propios repasos — una información, nunca un mando: blunderDB no modifica su objetivo para perseguir su tasa de acierto. Por debajo de una veintena de repasos, la medida no se muestra: se leería como un hecho cuando sólo es ruido.</p>
<p>Cambiar la retención <strong>no es retroactivo</strong>: cada carta adopta el nuevo ritmo en su próximo repaso, y los vencimientos ya fijados no se mueven. El efecto es, por tanto, progresivo e invisible el mismo día.</p>
<p>El <em>intervalo máximo</em> acota el espaciado. Un mazo creado recientemente arranca en un año: una posición que el algoritmo aplazaría varios años ha abandonado el mazo sin que usted lo haya decidido, y su propio juego cambia más deprisa que eso. Los mazos más antiguos conservan el valor que tenían.</p>
<h3>Microentrenamientos</h3>
<p>El panel Anki hace repasar un <strong>juicio</strong>; los microentrenamientos trabajan los tres <strong>cálculos</strong> que se hacen en la mesa, con el reloj, y que ninguna repetición espaciada desarrolla. El comando <code>train</code> inicia una sesión de cinco preguntas:</p>
<ul>
<li><code>train pips</code> — contar los pips del jugador que mueve, sobre la posición mostrada.</li>
<li><code>train epc</code> — estimar el EPC de ese mismo jugador, sobre una posición de carrera que el motor sabe evaluar.</li>
<li><code>train tp</code> — recordar el punto de aceptación de una carrera larga en un marcador sorteado al azar, el de la tabla <code>tp2_live</code>.</li>
</ul>
<p>La pregunta ES la posición mostrada: el tablero es el de la aplicación, y la barra superior solo lleva la pregunta, la entrada y la corrección. La respuesta se escribe y se valida con el teclado (<em>Intro</em> comprueba y luego pasa a la siguiente; <em>Esc</em> abandona la sesión).</p>
<p>La tolerancia depende del ejercicio, y se declara en vez de adivinarse: el recuento de pips no tiene <strong>ninguna</strong> — una suma exacta salvo un pip es una suma equivocada — el EPC acepta medio pip, el punto de aceptación dos puntos porcentuales. Al final, la sesión muestra el número de aciertos y el tiempo <strong>mediano</strong> por pregunta.</p>
<p>Solo se conserva ese resumen, en los metadatos de la base: la sesión no guarda rastro pregunta por pregunta, y no se escribe nada hasta que termina. Salir a mitad de camino, por tanto, no registra nada.</p>
<h4>Cuestionario: el PR de entrenamiento</h4>
<p><code>train quiz</code> plantea un cuarto tipo de pregunta. El panel Anki hace memorizar; el cuestionario <strong>pone a prueba</strong>. Se extraen cinco posiciones ya analizadas de la lista recorrida, y hay que decidir:</p>
<ul>
<li>en una decisión de fichas, escribir la jugada con el teclado, en notación (<code>13/7 8/7</code>);</li>
<li>en una decisión de cubo, pulsar <em>Sin doblar</em>, <em>Doblar, aceptar</em> o <em>Doblar, pasar</em>.</li>
</ul>
<p>El panel Análisis queda tapado mientras la pregunta no tenga respuesta: lleva la respuesta, y una pregunta cuya respuesta se muestra al lado no es una pregunta.</p>
<p>La corrección distingue tres desenlaces, y confundirlos mentiría. Una <strong>jugada ilegal</strong> no es una jugada mal elegida: es un error de reglas. Una <strong>jugada legal que el motor no clasificó</strong> no es un error en absoluto: simplemente no tiene precio, y no le cuesta nada a la sesión. Una jugada clasificada cuesta lo que el análisis dice que cuesta, en milipuntos.</p>
<p>Al final, la sesión muestra un <strong>PR de cuestionario</strong> calculado con la fórmula que las estadísticas aplican al juego real: 500 × error medio en equidad normalizada. Eso hace comparables ambos números: un PR de cuestionario de 6 y un PR de partido de 6 miden lo mismo en la misma escala.</p>
<h3>Panel de Metadatos</h3>
<p>El panel <strong>Metadatos</strong> muestra la información general de la base de datos actual: nombre, descripción, número de posiciones, número de partidas y juegos, versión del esquema. Accesible mediante el comando <code>meta</code>.</p>
<p>También muestra, <strong>cuando existe</strong>, el origen de la base de datos — véase Distribuir una base de datos: origen y contraseña. Una base de datos corriente no muestra esa sección.</p>
<h3>Distribuir una base de datos: origen y contraseña</h3>
<p>Un profesor que distribuye una base de posiciones dispone de dos mecanismos, independientes entre sí, ambos opcionales y elegidos <strong>en el momento de la exportación</strong>: marcar el archivo con su origen y protegerlo con una contraseña.</p>
<div class="admonition note">
<p>Ninguno de los dos hace seguimiento de lo que ocurre con el archivo. blunderDB <strong>no registra nada del lado de quien recibe la base</strong>: abrir una base marcada es exactamente igual que abrir cualquier otra, y en ningún sitio queda constancia de quién la abrió, cuándo, ni de dónde procede su contenido.</p>
</div>
<h4>Marcar una base de datos con su origen</h4>
<p>La ventana de exportación cabe en una sola pantalla: el formulario y, sobre él, una barra de progreso durante la escritura. Se cierra sola al terminar y el resultado aparece en la barra de estado.</p>
<p>Tres puntos merecen atención:</p>
<ul>
<li><strong>La exportación afecta a las posiciones mostradas actualmente</strong>, no a toda la base. Tras una búsqueda solo se exportan los resultados: la ventana lo recuerda en la parte superior.</li>
<li><strong>Una colección cuyas posiciones no estén todas en la selección llega truncada.</strong> Por eso la lista muestra, para cada colección, la parte cubierta («12/40») y la señala en rojo cuando es parcial.</li>
<li><strong>Los torneos solo pueden exportarse junto con los partidos</strong>: sin ellos no existe el vínculo torneo–partido y el torneo llegaría vacío. La casilla permanece desactivada mientras no se marque «incluir los partidos».</li>
</ul>
<p>Los campos <em>Usuario</em>, <em>Descripción</em> y <em>Fecha</em> describen el <strong>archivo producido</strong>; se rellenan previamente a partir de la base de origen. La casilla <em>Mis filtros guardados</em> es distinta de las demás: no exporta contenido, sino tus propias búsquedas guardadas, que no sirven de nada en la base de otra persona.</p>
<p>Al marcar <strong>Marcar este archivo con su origen</strong> aparecen dos campos:</p>
<ul>
<li><strong>Origen</strong> — qué es este archivo y de dónde viene, con tus palabras: «Clase de Jean Dupont — 12 de marzo de 2026». Este campo es <strong>obligatorio</strong>: mientras esté vacío, el botón de exportación permanece inactivo.</li>
<li><strong>Nota</strong>, opcional — condiciones de uso, dirección de contacto, una petición de no redistribuir.</li>
</ul>
<p>La marca se firma con tu identidad de emisor. Es por tanto <strong>inalterable e infalsificable</strong>: nadie puede modificarla ni fabricar una en tu nombre. En cambio, <strong>no es imborrable</strong>: el archivo distribuido es una base SQLite corriente y blunderDB es software libre. No impide nada: dice de dónde viene el archivo.</p>
<h4>Identidad del emisor</h4>
<p>Las marcas se firman con tu <strong>identidad de emisor</strong>, creada por sí sola la primera vez que marcas un archivo; no hay nada que configurar. Pertenece a una persona y no a una base de datos: todos tus archivos llevan la misma huella pública, con la forma <code>A3F1-9C24-7B05-E1D8</code>.</p>
<p>Puedes comunicar esa huella a tus destinatarios para que comprueben que un archivo procede realmente de ti. La identidad se traslada de un equipo a otro en un único archivo (extensión <code>.bdbid</code>), protegido si se desea por una frase de contraseña. <strong>Ese archivo permite firmar en tu nombre: no lo compartas.</strong></p>
<p>En los ajustes (icono de rueda dentada de la barra de herramientas), la pestaña <em>Identidad del emisor</em> muestra tu nombre y tu huella, y ofrece <em>Guardar identidad…</em>, <em>Cargar identidad…</em> y <em>Regenerar…</em>.</p>
<div class="admonition warning">
<p><strong>Regenerar no revoca nada.</strong> Una marca lleva incorporada la clave pública que la firmó: se verifica por sí sola para siempre. Si tu archivo de identidad se ha filtrado, quien lo posea podrá seguir firmando con tu huella antigua, y esas marcas seguirán siendo válidas.</p>
<p>Lo que te protege tras una filtración no es el software: es publicar tu nueva huella y desautorizar la antigua ante tus destinatarios.</p>
<p>La regeneración sobrescribe la clave actual; blunderDB ofrece guardarla antes de reemplazarla.</p>
</div>
<h4>Proteger una base de datos con una contraseña</h4>
<p>La contraseña se escribe oculta, tanto aquí como al abrir un archivo protegido; el icono con forma de ojo la muestra <strong>mientras se mantiene pulsado</strong> y vuelve a ocultarla al soltarlo.</p>
<p>Marcar <strong>Proteger este archivo con una contraseña</strong> produce un archivo con extensión <code>.dbx</code>, incluso si habías elegido un nombre en <code>.db</code> en la ventana de guardado, ya que esta se abre antes de pedir la contraseña. Para abrirlo, usa la apertura de base habitual: la ventana de selección acepta tanto <code>.db</code> como <code>.dbx</code>. blunderDB pide entonces la contraseña e instala al lado una base corriente; después no se pide nada más.</p>
<p>La ventana ofrece <strong>eliminar el archivo protegido una vez abierto</strong>: de lo contrario conservas el mismo contenido con dos nombres. La casilla no está marcada por omisión —el archivo protegido sigue siendo tuyo si piensas transmitirlo— y la eliminación solo se produce tras una apertura correcta.</p>
<div class="admonition warning">
<p>La contraseña protege el <strong>transporte</strong> del archivo, no la base. Impide que un tercero abra un archivo olvidado en una carpeta de descargas o un adjunto reenviado por error. No protege frente a aquel a quien has dado la contraseña.</p>
</div>
<p>La contraseña se comprueba en <strong>cada</strong> apertura, incluso si el archivo ya se había abierto antes en ese equipo.</p>
<p>Técnicamente, la base se cifra con <strong>AES-256 en modo GCM</strong>, con una clave derivada de la contraseña mediante <strong>Argon2id</strong> (64 MiB de memoria, 3 pasadas, 4 hilos) y una sal aleatoria propia de cada archivo. El modo GCM autentica el conjunto: una contraseña errónea se detecta como tal, y también cualquier alteración del archivo cifrado; nunca se obtiene en silencio una base corrupta.</p>
<p>La cabecera del archivo protegido permanece <strong>en claro</strong>: su origen sigue siendo legible sin la contraseña.</p>
<h4>Leer el origen de un archivo</h4>
<p>En la aplicación, abre el archivo y muestra el panel <strong>Metadatos</strong> (comando <code>meta</code>). Aparece una sección <strong>Origen</strong> en la parte superior del panel, de solo lectura, que indica lo que se inscribió, por quién, cuándo, y el estado de la firma:</p>
<ul>
<li>«✓ firma verificada — marcada por ti»: el archivo lleva tu marca, intacta;</li>
<li>«✓ firma verificada»: la marca está intacta y procede de otra clave; compara su huella con la que te haya comunicado el autor;</li>
<li>«⚠ firma no válida»: el documento ha sido modificado o falsificado.</li>
</ul>
<p>Esta sección no aparece en una base de datos corriente.</p>
<p>En línea de comandos, <code>blunderdb info --db archivo.db</code> muestra el origen y el estado de la firma, <strong>sin escribir nunca en el archivo</strong>. El comando funciona también con un archivo protegido, sin la contraseña. Consulta <code>CLI_USAGE.md</code> para las opciones <code>--watermark</code> y <code>--password</code> de <code>export</code>, así como para <code>identity</code> y <code>open</code>.</p>
<h4>Publicar una base para otros</h4>
<p>Una base marcada se distribuye como cualquier archivo — correo, sitio personal, memoria USB. blunderDB <strong>no ofrece ningún servicio</strong>: ni repositorio, ni catálogo alojado, ni cuenta. Es consecuencia directa de su diseño: nunca se registra nada del lado de quien recibe un archivo, así que no habría nada que comunicar a un servicio, aunque existiera.</p>
<p>Lo que hace utilizable por otro una base publicada se reduce a cuatro campos, todos ya presentes:</p>
<ul>
<li><strong>Usuario</strong> — quién la constituyó, con el nombre que quiere ver citado.</li>
<li><strong>Descripción</strong> — qué contiene la base, en una frase que quepa en una lista: «240 decisiones de cubo al marcador, comentadas, nivel intermedio».</li>
<li><strong>Origen</strong> (de la marca de agua) — qué es este archivo y para quién se produjo. Es lo primero que el destinatario lee en el panel <em>Metadatos</em>.</li>
<li><strong>Huella del emisor</strong> — publíquela junto al archivo, no dentro: comparándola el destinatario comprueba que el archivo viene de usted y no de alguien que ha tomado su nombre.</li>
</ul>
<p>Una base publicada sin marca de agua sigue siendo perfectamente utilizable; simplemente es anónima, y el panel <em>Metadatos</em> no muestra entonces ninguna sección <em>Origen</em>.</p>
<p>Para dar a conocer una base, la categoría <em>Show and tell</em> de las <code>discusiones del repositorio &lt;https://github.com/kevung/blunderDB/discussions&gt;</code>_ sirve de directorio: es una lista mantenida por quienes publican, no un servicio prestado por blunderDB. Anunciar una allí requiere el enlace, los cuatro campos anteriores y la huella.</p>
`,
    shortcuts: `
<h3>Base de datos</h3>
<table>
<thead>
<tr>
<th>Atajo</th>
<th>Acción</th>
</tr>
</thead>
<tbody>
<tr>
<td>CTRL-N</td>
<td>Crear una nueva base de datos.</td>
</tr>
<tr>
<td>CTRL-O</td>
<td>Abrir una base de datos existente.</td>
</tr>
<tr>
<td>CTRL-MAYÚS-I</td>
<td>Fusionar una base de datos en esta.</td>
</tr>
<tr>
<td>CTRL-MAYÚS-S</td>
<td>Exportar la base de datos.</td>
</tr>
<tr>
<td>CTRL-Q</td>
<td>Cerrar blunderDB.</td>
</tr>
<tr>
<td>CTRL-M</td>
<td>Editar los metadatos de la base de datos.</td>
</tr>
</tbody>
</table>
<h3>Posición</h3>
<table>
<thead>
<tr>
<th>Atajo</th>
<th>Acción</th>
</tr>
</thead>
<tbody>
<tr>
<td>CTRL-I</td>
<td>Importar una o varias posiciones/partidas desde un archivo (xg, xgp, sgf, mat, txt, bgf).</td>
</tr>
<tr>
<td>CTRL-MAYÚS-F</td>
<td>Importar recursivamente una carpeta de archivos de partidas/posiciones.</td>
</tr>
<tr>
<td>CTRL-C</td>
<td>Copiar una posición al portapapeles.</td>
</tr>
<tr>
<td>CTRL-X</td>
<td>Copiar la imagen del tablero al portapapeles (PNG).</td>
</tr>
<tr>
<td>CTRL-X CTRL-X</td>
<td>Copiar la imagen del tablero con el análisis al portapapeles (PNG).</td>
</tr>
<tr>
<td>CTRL-V</td>
<td>Pegar una posición desde el portapapeles (detección automática del formato).</td>
</tr>
<tr>
<td>CTRL-S</td>
<td>Guardar una posición.</td>
</tr>
<tr>
<td>CTRL-U</td>
<td>Actualizar una posición.</td>
</tr>
<tr>
<td>Supr</td>
<td>Eliminar la posición actual (se pide confirmación).</td>
</tr>
<tr>
<td>RETROCESO</td>
<td>Reiniciar el tablero, el cubo, el marcador y los dados.</td>
</tr>
<tr>
<td>CTRL-G</td>
<td>Mostrar los metadatos de la posición.</td>
</tr>
</tbody>
</table>
<h3>Navegación</h3>
<table>
<thead>
<tr>
<th>Atajo</th>
<th>Acción</th>
</tr>
</thead>
<tbody>
<tr>
<td>CTRL-R</td>
<td>Recargar todas las posiciones de la base de datos.</td>
</tr>
<tr>
<td>AvPág, h</td>
<td>Primera posición / Partida anterior (navegación de partida).</td>
</tr>
<tr>
<td>IZQUIERDA, k</td>
<td>Posición anterior.</td>
</tr>
<tr>
<td>DERECHA, j</td>
<td>Posición siguiente.</td>
</tr>
<tr>
<td>ARRIBA, k</td>
<td>Jugada anterior (cuando hay una jugada seleccionada en el análisis).</td>
</tr>
<tr>
<td>ABAJO, j</td>
<td>Jugada siguiente (cuando hay una jugada seleccionada en el análisis).</td>
</tr>
<tr>
<td>RePág, l</td>
<td>Última posición / Partida siguiente (navegación de partida).</td>
</tr>
<tr>
<td>r</td>
<td>Cargar una posición aleatoria.</td>
</tr>
</tbody>
</table>
<h3>Visualización</h3>
<table>
<thead>
<tr>
<th>Atajo</th>
<th>Acción</th>
</tr>
</thead>
<tbody>
<tr>
<td>CTRL-IZQUIERDA</td>
<td>Orientación del tablero hacia la izquierda.</td>
</tr>
<tr>
<td>CTRL-DERECHA</td>
<td>Orientación del tablero hacia la derecha.</td>
</tr>
<tr>
<td>p</td>
<td>Mostrar/ocultar el recuento de pips.</td>
</tr>
</tbody>
</table>
<h3>Acciones</h3>
<table>
<thead>
<tr>
<th>Atajo</th>
<th>Acción</th>
</tr>
</thead>
<tbody>
<tr>
<td>TAB</td>
<td>Abrir el panel de búsqueda (editor de posiciones).</td>
</tr>
<tr>
<td>ESPACIO</td>
<td>Abrir la línea de comandos.</td>
</tr>
</tbody>
</table>
<h3>Herramientas</h3>
<table>
<thead>
<tr>
<th>Atajo</th>
<th>Acción</th>
</tr>
</thead>
<tbody>
<tr>
<td>CTRL-L</td>
<td>Mostrar/ocultar el análisis.</td>
</tr>
<tr>
<td>CTRL-P</td>
<td>Mostrar/ocultar los comentarios.</td>
</tr>
<tr>
<td>CTRL-K</td>
<td>Mostrar/ocultar el panel Anki (repetición espaciada).</td>
</tr>
<tr>
<td>CTRL-F</td>
<td>Mostrar/ocultar el panel de búsqueda.</td>
</tr>
<tr>
<td>CTRL-Tab</td>
<td>Mostrar/ocultar el panel de partidas.</td>
</tr>
<tr>
<td>CTRL-B</td>
<td>Mostrar/ocultar el panel de colecciones.</td>
</tr>
<tr>
<td>CTRL-Y</td>
<td>Mostrar/ocultar el panel de torneos.</td>
</tr>
<tr>
<td>CTRL-D</td>
<td>Mostrar/ocultar el panel de estadísticas.</td>
</tr>
<tr>
<td>CTRL-E</td>
<td>Mostrar/ocultar el panel Eval.</td>
</tr>
<tr>
<td>?</td>
<td>Mostrar/ocultar la ayuda.</td>
</tr>
</tbody>
</table>
<h3>Pestañas de vistas</h3>
<table>
<thead>
<tr>
<th>Atajo</th>
<th>Acción</th>
</tr>
</thead>
<tbody>
<tr>
<td>CTRL-T</td>
<td>Crear una nueva vista (copia de la vista actual).</td>
</tr>
<tr>
<td>CTRL-W</td>
<td>Cerrar la vista actual.</td>
</tr>
<tr>
<td>CTRL-AvPág, MAYÚS-J</td>
<td>Vista anterior.</td>
</tr>
<tr>
<td>CTRL-RePág, MAYÚS-K</td>
<td>Vista siguiente.</td>
</tr>
<tr>
<td>CTRL-1 … CTRL-9</td>
<td>Ir directamente a la n-ésima vista.</td>
</tr>
<tr>
<td>Doble clic en la pestaña</td>
<td>Renombrar la vista.</td>
</tr>
</tbody>
</table>
<h3>Línea de comandos</h3>
<table>
<thead>
<tr>
<th>Atajo</th>
<th>Acción</th>
</tr>
</thead>
<tbody>
<tr>
<td>ARRIBA</td>
<td>Recorrer el historial de comandos hacia arriba.</td>
</tr>
<tr>
<td>ABAJO</td>
<td>Recorrer el historial de comandos hacia abajo.</td>
</tr>
</tbody>
</table>
<h3>Historial de búsqueda</h3>
<table>
<thead>
<tr>
<th>Atajo</th>
<th>Acción</th>
</tr>
</thead>
<tbody>
<tr>
<td>Clic</td>
<td>Seleccionar/deseleccionar una búsqueda (mostrar la posición).</td>
</tr>
<tr>
<td>Doble clic</td>
<td>Ejecutar la búsqueda.</td>
</tr>
</tbody>
</table>
<h3>Biblioteca de filtros</h3>
<table>
<thead>
<tr>
<th>Atajo</th>
<th>Acción</th>
</tr>
</thead>
<tbody>
<tr>
<td>Clic</td>
<td>Seleccionar/deseleccionar un filtro (mostrar la posición).</td>
</tr>
<tr>
<td>Doble clic</td>
<td>Ejecutar la búsqueda del filtro.</td>
</tr>
</tbody>
</table>
<h3>Panel de análisis</h3>
<table>
<thead>
<tr>
<th>Atajo</th>
<th>Acción</th>
</tr>
</thead>
<tbody>
<tr>
<td>Clic</td>
<td>Seleccionar/deseleccionar una jugada (mostrar/ocultar las flechas).</td>
</tr>
<tr>
<td>ARRIBA, k</td>
<td>Seleccionar la jugada anterior (cuando hay una jugada seleccionada).</td>
</tr>
<tr>
<td>ABAJO, j</td>
<td>Seleccionar la jugada siguiente (cuando hay una jugada seleccionada).</td>
</tr>
<tr>
<td>d</td>
<td>Alternar entre el análisis de jugadas y de cubo (solo en navegación de partida).</td>
</tr>
<tr>
<td>Esc</td>
<td>Deseleccionar la jugada. Si no hay ninguna jugada seleccionada, cerrar el panel.</td>
</tr>
</tbody>
</table>
<h3>Panel Eval</h3>
<table>
<thead>
<tr>
<th>Atajo</th>
<th>Acción</th>
</tr>
</thead>
<tbody>
<tr>
<td>Clic</td>
<td>Seleccionar/deseleccionar una jugada (mostrar/ocultar las flechas).</td>
</tr>
<tr>
<td>ARRIBA, k</td>
<td>Seleccionar la jugada anterior (cuando hay una jugada seleccionada).</td>
</tr>
<tr>
<td>ABAJO, j</td>
<td>Seleccionar la jugada siguiente (cuando hay una jugada seleccionada).</td>
</tr>
<tr>
<td>Esc</td>
<td>Deseleccionar la jugada.</td>
</tr>
</tbody>
</table>
<h3>Panel de partidas</h3>
<table>
<thead>
<tr>
<th>Atajo</th>
<th>Acción</th>
</tr>
</thead>
<tbody>
<tr>
<td>Clic</td>
<td>Seleccionar una partida.</td>
</tr>
<tr>
<td>Doble clic</td>
<td>Navegar por la partida.</td>
</tr>
<tr>
<td>ARRIBA, k</td>
<td>Seleccionar la partida anterior.</td>
</tr>
<tr>
<td>ABAJO, j</td>
<td>Seleccionar la partida siguiente.</td>
</tr>
<tr>
<td>INTRO</td>
<td>Cargar la partida seleccionada.</td>
</tr>
<tr>
<td>Supr</td>
<td>Eliminar la partida seleccionada.</td>
</tr>
<tr>
<td>Esc</td>
<td>Deseleccionar/cerrar el panel.</td>
</tr>
</tbody>
</table>
<h3>Panel Anki (repetición espaciada)</h3>
<table>
<thead>
<tr>
<th>Atajo</th>
<th>Acción</th>
</tr>
</thead>
<tbody>
<tr>
<td>ESPACIO, Clic</td>
<td>Mostrar la respuesta (el análisis registrado de la posición).</td>
</tr>
<tr>
<td>1</td>
<td>Calificar: Repetir (fallo, revisar pronto).</td>
</tr>
<tr>
<td>2</td>
<td>Calificar: Difícil.</td>
</tr>
<tr>
<td>3</td>
<td>Calificar: Bien.</td>
</tr>
<tr>
<td>4</td>
<td>Calificar: Fácil.</td>
</tr>
<tr>
<td>p</td>
<td>Mostrar/ocultar el conteo de pips (igual que el atajo general, disponible durante el repaso).</td>
</tr>
<tr>
<td>Esc</td>
<td>Detener la revisión y volver a la lista de mazos (se puede reanudar más tarde).</td>
</tr>
</tbody>
</table>
<h3>Panel de torneos</h3>
<table>
<thead>
<tr>
<th>Atajo</th>
<th>Acción</th>
</tr>
</thead>
<tbody>
<tr>
<td>Clic, Doble clic</td>
<td>Seleccionar un torneo (mostrar su detalle).</td>
</tr>
<tr>
<td>ARRIBA, k</td>
<td>Seleccionar el torneo anterior.</td>
</tr>
<tr>
<td>ABAJO, j</td>
<td>Seleccionar el torneo siguiente.</td>
</tr>
<tr>
<td>Doble clic (sobre una partida del torneo)</td>
<td>Navegar por la partida.</td>
</tr>
<tr>
<td>Esc</td>
<td>Cancelar la edición en curso, si no borrar la búsqueda de añadir partida, si no deseleccionar el torneo, si no cerrar el panel (por etapas).</td>
</tr>
</tbody>
</table>
<h3>Panel de colecciones</h3>
<table>
<thead>
<tr>
<th>Atajo</th>
<th>Acción</th>
</tr>
</thead>
<tbody>
<tr>
<td>Clic</td>
<td>Añadir/quitar la posición actual de la colección señalada por el cursor.</td>
</tr>
<tr>
<td>Doble clic</td>
<td>Abrir la colección.</td>
</tr>
<tr>
<td>Supr</td>
<td>Quitar la posición actual (o las posiciones marcadas) de la colección abierta.</td>
</tr>
<tr>
<td>Esc</td>
<td>Volver a la lista de colecciones, si no deseleccionar la colección, si no cerrar el panel (por etapas).</td>
</tr>
</tbody>
</table>
<h3>Panel de ayuda</h3>
<table>
<thead>
<tr>
<th>Atajo</th>
<th>Acción</th>
</tr>
</thead>
<tbody>
<tr>
<td>IZQUIERDA, h</td>
<td>Pestaña anterior.</td>
</tr>
<tr>
<td>DERECHA, l</td>
<td>Pestaña siguiente.</td>
</tr>
<tr>
<td>ARRIBA, k</td>
<td>Desplazar hacia arriba.</td>
</tr>
<tr>
<td>ABAJO, j</td>
<td>Desplazar hacia abajo.</td>
</tr>
<tr>
<td>ESPACIO</td>
<td>Página siguiente.</td>
</tr>
<tr>
<td>AvPág</td>
<td>Principio del contenido.</td>
</tr>
<tr>
<td>RePág</td>
<td>Final del contenido.</td>
</tr>
<tr>
<td>?, CTRL-F, Esc</td>
<td>Cerrar la ayuda.</td>
</tr>
</tbody>
</table>
`,
    commands: `
<p>La línea de comandos, situada en la barra de estado, se abre pulsando la tecla <em>ESPACIO</em>. Al escribir un comando, aparece automáticamente una lista de sugerencias: la tecla <em>TAB</em> (o <em>MAYÚS-TAB</em>) recorre las propuestas y completa el comando, mientras que <em>ESC</em> cierra la lista (un segundo <em>ESC</em> cierra la línea de comandos). Las teclas <em>ARRIBA</em> y <em>ABAJO</em> siguen reservadas al historial de comandos.</p>
<h3>Operaciones globales</h3>
<table>
<thead>
<tr>
<th>Comando</th>
<th>Acción</th>
</tr>
</thead>
<tbody>
<tr>
<td>new, ne, n</td>
<td>Crea una nueva base de datos.</td>
</tr>
<tr>
<td>open, op, o</td>
<td>Abre una base de datos existente.</td>
</tr>
<tr>
<td>import_db, idb</td>
<td>Importa y fusiona otra base de datos.</td>
</tr>
<tr>
<td>export_db, edb</td>
<td>Exporta la selección actual a una nueva base de datos.</td>
</tr>
<tr>
<td>quit, q</td>
<td>Cierra blunderDB.</td>
</tr>
<tr>
<td>help, he, h</td>
<td>Abre la ayuda de blunderDB.</td>
</tr>
<tr>
<td>tutorial, tour</td>
<td>Abre el catálogo de visitas guiadas de la interfaz.</td>
</tr>
<tr>
<td>demo</td>
<td>Carga una base de ejemplo (partidas, torneo, colecciones, comentarios, mazo Anki, análisis) para descubrir la herramienta.</td>
</tr>
<tr>
<td>meta</td>
<td>Muestra los metadatos de la base de datos.</td>
</tr>
<tr>
<td>epc</td>
<td>Abre el panel Eval (Effective Pip Count, probabilidad de victoria y veredicto de cubo en bearoff). <code>epc</code> es el antiguo nombre de este panel, conservado.</td>
</tr>
<tr>
<td>met</td>
<td>Abre la tabla de equidad de match Kazaross-XG2.</td>
</tr>
<tr>
<td>cm</td>
<td>Abre la matriz del cubo: el veredicto de la posición actual en todos los marcadores de un partido a 5, 7 o 9 puntos.</td>
</tr>
<tr>
<td>tags</td>
<td>Abre el vocabulario de etiquetas: las etiquetas usadas en esta base, con el número de posiciones, pulsables para lanzar la búsqueda.</td>
</tr>
<tr>
<td>log</td>
<td>Abre el registro de actividad: las últimas doscientas líneas del archivo de registro, con lo necesario para copiarlas en un informe o abrir la carpeta que las contiene.</td>
</tr>
<tr>
<td>ask</td>
<td>Traduce una frase en palabras — francés o inglés — a tokens de búsqueda: <code>ask my cube blunders at a score</code>. Los tokens se escriben en la barra de comandos, no se ejecutan: se releen y luego Intro. Lo que no se ha entendido se dice, nunca se adivina.</td>
</tr>
<tr>
<td>like</td>
<td>Sustituye la lista recorrida por las posiciones más cercanas a la actual — o a aquella cuyo índice se indica (<code>like 42</code>). La cercanía es una distancia de transporte en pips de ficha: no es un filtro, ordena toda la base en vez de restringirla, y por tanto no se combina con los tokens de búsqueda.</td>
</tr>
<tr>
<td>train</td>
<td>Inicia una sesión de microentrenamiento. Toma un argumento: <code>train pips</code> (recuento de pips), <code>train epc</code>, <code>train tp</code> (punto de aceptación al marcador), <code>train quiz</code> (la jugada o la acción de cubo, calificadas contra el análisis guardado). Cinco preguntas, cronometradas, corregidas al instante.</td>
</tr>
<tr>
<td>tp2</td>
<td>Abre la tabla de takepoints con el cubo a 2.</td>
</tr>
<tr>
<td>tp2_live</td>
<td>Abre la tabla de takepoints con el cubo a 2 para carreras largas.</td>
</tr>
<tr>
<td>tp2_last</td>
<td>Abre la tabla de takepoints con el cubo a 2 muerto.</td>
</tr>
<tr>
<td>tp4</td>
<td>Abre la tabla de takepoints con el cubo a 4.</td>
</tr>
<tr>
<td>tp4_live</td>
<td>Abre la tabla de takepoints con el cubo a 4 para carreras largas.</td>
</tr>
<tr>
<td>tp4_last</td>
<td>Abre la tabla de takepoints con el cubo a 4 muerto.</td>
</tr>
<tr>
<td>gv1</td>
<td>Abre la tabla de valores de gammon con el cubo a 1.</td>
</tr>
<tr>
<td>gv2</td>
<td>Abre la tabla de valores de gammon con el cubo a 2.</td>
</tr>
<tr>
<td>gv4</td>
<td>Abre la tabla de valores de gammon con el cubo a 4.</td>
</tr>
</tbody>
</table>
<h3>Posiciones y navegación</h3>
<table>
<thead>
<tr>
<th>Comando</th>
<th>Acción</th>
</tr>
</thead>
<tbody>
<tr>
<td>import, i</td>
<td>Importa una o varias posiciones/partidos desde un fichero (xg, xgp, sgf, mat, txt, bgf). Con un argumento — <code>import XGID=…</code> o <code>import OGID=…</code> — lee el identificador en vez de abrir un selector de ficheros, para cuando llega de un mensaje, un foro o un script.</td>
</tr>
<tr>
<td>delete, del, d</td>
<td>Elimina la posición actual (se pide confirmación); el borrado pasa por la papelera y se puede deshacer durante treinta días.</td>
</tr>
<tr>
<td>trash</td>
<td>Abre la papelera: lo que se ha eliminado, con lo necesario para restaurarlo.</td>
</tr>
<tr>
<td>[number]</td>
<td>Ir a la posición del índice indicado.</td>
</tr>
<tr>
<td>list, l</td>
<td>Mostrar el análisis de la posición actual.</td>
</tr>
<tr>
<td>comment, co</td>
<td>Mostrar/escribir comentarios.</td>
</tr>
<tr>
<td>history, hi</td>
<td>Abrir el panel de búsqueda (el historial de búsqueda se encuentra en su pestaña <em>Historial</em>).</td>
</tr>
<tr>
<td>stats, st</td>
<td>Mostrar/ocultar el panel de estadísticas.</td>
</tr>
<tr>
<td>match, ma</td>
<td>Mostrar/ocultar el panel de matches.</td>
</tr>
<tr>
<td>collection, coll</td>
<td>Mostrar/ocultar el panel de colecciones.</td>
</tr>
<tr>
<td>#tag1 tag2 ...</td>
<td>Etiquetar la posición actual.</td>
</tr>
<tr>
<td>e</td>
<td>Cargar todas las posiciones de la base de datos.</td>
</tr>
<tr>
<td>blunders, bl [n]</td>
<td>Carga los peores errores (equity/MWC) en la vista de análisis, según el filtro de estadísticas actual. Un número opcional elige cuántos cargar (<code>bl 50</code>); 10 por defecto.Carga los peores errores (equity/MWC) en la vista de análisis, según el filtro de estadísticas actual.</td>
</tr>
<tr>
<td>m</td>
<td>Navegar por el último match visitado.</td>
</tr>
</tbody>
</table>
<h3>Edición y búsqueda</h3>
<table>
<thead>
<tr>
<th>Comando</th>
<th>Acción</th>
</tr>
</thead>
<tbody>
<tr>
<td>write, wr, w</td>
<td>Guarda la posición actual.</td>
</tr>
<tr>
<td>write!, wr!, w!</td>
<td>Actualizar la posición actual.</td>
</tr>
<tr>
<td>s</td>
<td>Buscar posiciones con filtros.</td>
</tr>
<tr>
<td>ss</td>
<td>Buscar entre las posiciones actualmente filtradas.</td>
</tr>
</tbody>
</table>
<h3>Filtros de búsqueda</h3>
<p>Esta tabla es la referencia de la gramática de búsqueda: la línea de comandos, la biblioteca de filtros y la opción <code>--query</code> de <code>blunderdb search</code> leen todos los mismos tokens. La columna <em>Equivalente CLI</em> da, cuando existe, la opción de <code>search</code> que hace lo mismo (ver Interfaz de línea de comandos (CLI)); un guion señala un filtro que solo la gramática expresa.</p>
<p>Cinco tokens no llevan su valor: lo leen en el tablero de búsqueda. <code>cube</code> y <code>score</code> retoman el cubo y el score puestos en él, <code>d</code> el tipo de decisión, <code>D</code> y <code>D1</code> los dados, <code>x</code> la estructura dibujada en la pestaña <em>Excepto</em>. Una tirada nunca se escribe entonces en el token: <code>D65</code> no existe, solo la forma de exclusión lleva sus cifras (<code>xD65</code>). En la línea de comandos, donde no hay tablero, estos tokens se comparan con un tablero vacío; son las opciones de la tercera columna las que hay que emplear ahí.</p>
<p>Los errores y las equidades se cuentan en <strong>milésimas de equidad</strong> — los <em>milipuntos</em> de la tabla siguiente: <code>E&gt;100</code> conserva las jugadas que costaron al menos una décima de punto, valiendo un punto 1000 milésimas.</p>
<p>Dos búsquedas completas:</p>
<ul>
<li><code>s p&gt;30 w40,60 xco</code> — más de 30 pips de retraso, entre 40 % y 60 % de probabilidad de victoria, sin comentario.</li>
<li><code>s ph:race E&gt;50 co:xg</code> — en carrera, una jugada que costó al menos 50 milésimas, y un comentario venido de eXtreme Gammon.</li>
</ul>
<table>
<thead>
<tr>
<th>Consulta</th>
<th>Acción</th>
<th>Equivalente CLI</th>
</tr>
</thead>
<tbody>
<tr>
<td>cube, cub, cu, c</td>
<td>La posición cumple la configuración del cubo.</td>
<td><code>--cube</code></td>
</tr>
<tr>
<td>score, sco, sc, s</td>
<td>La posición cumple el marcador.</td>
<td><code>--score1</code> <code>--score2</code></td>
</tr>
<tr>
<td>d</td>
<td>La posición cumple el tipo de decisión (ficha o cubo).</td>
<td><code>--decision</code></td>
</tr>
<tr>
<td>D</td>
<td>La posición cumple la tirada de dados (ambos dados, sin importar el orden).</td>
<td><code>--dice 6,5</code></td>
</tr>
<tr>
<td>D1</td>
<td>La posición cumple la tirada de dados únicamente en el primer dado (el valor del primer dado aparece en cualquiera de los dos dados de la posición).</td>
<td><code>--dice 6</code></td>
</tr>
<tr>
<td>xD65</td>
<td>La posición <strong>no</strong> se jugó con la tirada 6-5 (sin importar el orden). El valor se indica en el token; repetible para excluir varias tiradas (<code>xD65 xD54</code>).</td>
<td>—</td>
</tr>
<tr>
<td>nc</td>
<td>La posición es sin contacto.</td>
<td>—</td>
</tr>
<tr>
<td>ph:race</td>
<td>La posición se encuentra en una fase de juego dada: <code>opening</code> (apertura), <code>middlegame</code> (medio juego), <code>race</code> (carrera) o <code>bearoff</code> (retirada de fichas). Repetible (<code>ph:race ph:bearoff</code>). La etiqueta se deriva del tablero y nunca es editable; <code>blunderdb repair</code> la vuelve a calcular.</td>
<td><code>--phase</code></td>
</tr>
<tr>
<td>gt:holding</td>
<td>La posición corresponde a un plan de juego dado, desde el punto de vista del jugador que mueve: <code>race</code>, <code>bearin</code> (entrada bajo contacto), <code>crunch</code>, <code>backgame</code>, <code>acepoint</code>, <code>blitz</code>, <code>primevprime</code>, <code>mutualholding</code>, <code>holding</code>, <code>contact</code>. Repetible (<code>gt:holding gt:mutualholding</code>). Etiqueta derivada como la fase: calculada a partir del tablero, nunca editable, recalculada por <code>blunderdb repair</code>.</td>
<td><code>--game-type</code></td>
</tr>
<tr>
<td>#prime</td>
<td>La posición lleva esta <strong>etiqueta</strong> en uno de sus comentarios. Una etiqueta es una <code>#palabra</code> escrita en la prosa; nada la declara. La comparación es delimitada, así que <code>#prime</code> no encuentra <code>#priming</code> — esa es toda la diferencia con el filtro de texto, que busca una subcadena. Repetible, y las etiquetas se <strong>acumulan</strong> (<code>#prime #backgame</code> pide ambas): una posición lleva varias etiquetas, así que nombrar dos quiere decir «las dos».</td>
<td>—</td>
</tr>
<tr>
<td>n&gt;x</td>
<td>La posición se encontró más de x veces en la base — el número de jugadas que llegan a ella, en todos los partidos. Formas <code>n&gt;3</code>, <code>n&lt;2</code>, <code>n3,10</code> y <code>n4</code> (exactamente cuatro).</td>
<td>—</td>
</tr>
<tr>
<td>M</td>
<td>La posición o su réplica especular cumple los filtros.</td>
<td>—</td>
</tr>
<tr>
<td>i</td>
<td>La posición se importó por separado, y no la trajo la importación de una partida.</td>
<td><code>--individual</code></td>
</tr>
<tr>
<td>fl</td>
<td>La posición fue marcada en el programa de origen, al importar una partida de eXtreme Gammon.</td>
<td><code>--flagged</code></td>
</tr>
<tr>
<td>x</td>
<td>La posición no contiene ninguna ficha de la estructura de exclusión (pestaña <em>Excepto</em> del panel de búsqueda).</td>
<td>—</td>
</tr>
<tr>
<td>p&gt;x</td>
<td>El jugador tiene al menos x pips de desventaja en la carrera.</td>
<td><code>--pip-min</code></td>
</tr>
<tr>
<td>p&lt;x</td>
<td>El jugador tiene como máximo x pips de desventaja en la carrera.</td>
<td><code>--pip-max</code></td>
</tr>
<tr>
<td>px,y</td>
<td>El jugador tiene entre x e y pips de desventaja en la carrera.</td>
<td><code>--pip-min</code> <code>--pip-max</code></td>
</tr>
<tr>
<td>P&gt;x</td>
<td>El jugador tiene una carrera de al menos x pips.</td>
<td>—</td>
</tr>
<tr>
<td>P&lt;x</td>
<td>El jugador tiene una carrera de como máximo x pips.</td>
<td>—</td>
</tr>
<tr>
<td>Px,y</td>
<td>El jugador tiene una carrera entre x e y pips.</td>
<td>—</td>
</tr>
<tr>
<td>e&gt;x</td>
<td>La equidad (en milipuntos) de la posición es mayor que x.</td>
<td>—</td>
</tr>
<tr>
<td>e&lt;x</td>
<td>La equidad (en milipuntos) de la posición es menor que x.</td>
<td>—</td>
</tr>
<tr>
<td>ex,y</td>
<td>La equidad (en milipuntos) de la posición está comprendida entre x e y.</td>
<td>—</td>
</tr>
<tr>
<td>E&gt;x</td>
<td>El error de la jugada realizada por el jugador 1 (en milipuntos) es mayor que x.</td>
<td><code>--move-error-min</code></td>
</tr>
<tr>
<td>E&lt;x</td>
<td>El error de la jugada realizada por el jugador 1 (en milipuntos) es menor que x.</td>
<td><code>--move-error-max</code></td>
</tr>
<tr>
<td>Ex,y</td>
<td>El error de la jugada realizada por el jugador 1 (en milipuntos) está comprendido entre x e y.</td>
<td><code>--move-error-min</code> <code>--move-error-max</code></td>
</tr>
<tr>
<td>w&gt;x</td>
<td>El jugador tiene probabilidades de victoria superiores al x %.</td>
<td><code>--winrate-min</code></td>
</tr>
<tr>
<td>w&lt;x</td>
<td>El jugador tiene probabilidades de victoria inferiores al x %.</td>
<td><code>--winrate-max</code></td>
</tr>
<tr>
<td>wx,y</td>
<td>El jugador tiene probabilidades de victoria entre el x % y el y %.</td>
<td><code>--winrate-min</code> <code>--winrate-max</code></td>
</tr>
<tr>
<td>g&gt;x</td>
<td>El jugador tiene probabilidades de gammon superiores al x %.</td>
<td>—</td>
</tr>
<tr>
<td>g&lt;x</td>
<td>El jugador tiene probabilidades de gammon inferiores al x %.</td>
<td>—</td>
</tr>
<tr>
<td>gx,y</td>
<td>El jugador tiene probabilidades de gammon entre el x % y el y %.</td>
<td>—</td>
</tr>
<tr>
<td>b&gt;x</td>
<td>El jugador tiene probabilidades de backgammon superiores al x %.</td>
<td>—</td>
</tr>
<tr>
<td>b&lt;x</td>
<td>El jugador tiene probabilidades de backgammon inferiores al x %.</td>
<td>—</td>
</tr>
<tr>
<td>bx,y</td>
<td>El jugador tiene probabilidades de backgammon entre el x % y el y %.</td>
<td>—</td>
</tr>
<tr>
<td>W&gt;x</td>
<td>El adversario tiene probabilidades de victoria superiores al x %.</td>
<td>—</td>
</tr>
<tr>
<td>W&lt;x</td>
<td>El adversario tiene probabilidades de victoria inferiores al x %.</td>
<td>—</td>
</tr>
<tr>
<td>Wx,y</td>
<td>El adversario tiene probabilidades de victoria entre el x % y el y %.</td>
<td>—</td>
</tr>
<tr>
<td>G&gt;x</td>
<td>El adversario tiene probabilidades de gammon superiores al x %.</td>
<td>—</td>
</tr>
<tr>
<td>G&lt;x</td>
<td>El adversario tiene probabilidades de gammon inferiores al x %.</td>
<td>—</td>
</tr>
<tr>
<td>Gx,y</td>
<td>El adversario tiene probabilidades de gammon entre el x % y el y %.</td>
<td>—</td>
</tr>
<tr>
<td>B&gt;x</td>
<td>El adversario tiene probabilidades de backgammon superiores al x %.</td>
<td>—</td>
</tr>
<tr>
<td>B&lt;x</td>
<td>El adversario tiene probabilidades de backgammon inferiores al x %.</td>
<td>—</td>
</tr>
<tr>
<td>Bx,y</td>
<td>El adversario tiene probabilidades de backgammon entre el x % y el y %.</td>
<td>—</td>
</tr>
<tr>
<td>o&gt;x</td>
<td>El jugador tiene al menos x fichas retiradas.</td>
<td><code>--off1-min</code></td>
</tr>
<tr>
<td>o&lt;x</td>
<td>El jugador tiene como máximo x fichas retiradas.</td>
<td>—</td>
</tr>
<tr>
<td>ox,y</td>
<td>El jugador tiene entre x e y fichas retiradas.</td>
<td>—</td>
</tr>
<tr>
<td>O&gt;x</td>
<td>El adversario tiene al menos x fichas retiradas.</td>
<td><code>--off2-min</code></td>
</tr>
<tr>
<td>O&lt;x</td>
<td>El adversario tiene como máximo x fichas retiradas.</td>
<td>—</td>
</tr>
<tr>
<td>Ox,y</td>
<td>El adversario tiene entre x e y fichas retiradas.</td>
<td>—</td>
</tr>
<tr>
<td>k&gt;x</td>
<td>El jugador tiene al menos x fichas rezagadas.</td>
<td>—</td>
</tr>
<tr>
<td>k&lt;x</td>
<td>El jugador tiene como máximo x fichas rezagadas.</td>
<td>—</td>
</tr>
<tr>
<td>kx,y</td>
<td>El jugador tiene entre x e y fichas rezagadas.</td>
<td>—</td>
</tr>
<tr>
<td>K&gt;x</td>
<td>El adversario tiene al menos x fichas rezagadas.</td>
<td>—</td>
</tr>
<tr>
<td>K&lt;x</td>
<td>El adversario tiene como máximo x fichas rezagadas.</td>
<td>—</td>
</tr>
<tr>
<td>Kx,y</td>
<td>El adversario tiene entre x e y fichas rezagadas.</td>
<td>—</td>
</tr>
<tr>
<td>z&gt;x</td>
<td>El jugador tiene al menos x fichas en la zona.</td>
<td>—</td>
</tr>
<tr>
<td>z&lt;x</td>
<td>El jugador tiene como máximo x fichas en la zona.</td>
<td>—</td>
</tr>
<tr>
<td>zx,y</td>
<td>El jugador tiene entre x e y fichas en la zona.</td>
<td>—</td>
</tr>
<tr>
<td>Z&gt;x</td>
<td>El adversario tiene al menos x fichas en la zona.</td>
<td>—</td>
</tr>
<tr>
<td>Z&lt;x</td>
<td>El adversario tiene como máximo x fichas en la zona.</td>
<td>—</td>
</tr>
<tr>
<td>Zx,y</td>
<td>El adversario tiene entre x e y fichas en la zona.</td>
<td>—</td>
</tr>
<tr>
<td>bo&gt;x</td>
<td>El jugador tiene al menos x blots en el outfield.</td>
<td>—</td>
</tr>
<tr>
<td>bo&lt;x</td>
<td>El jugador tiene como máximo x blots en el outfield.</td>
<td>—</td>
</tr>
<tr>
<td>box,y</td>
<td>El jugador tiene entre x e y blots en el outfield.</td>
<td>—</td>
</tr>
<tr>
<td>BO&gt;x</td>
<td>El adversario tiene al menos x blots en el outfield.</td>
<td>—</td>
</tr>
<tr>
<td>BO&lt;x</td>
<td>El adversario tiene como máximo x blots en el outfield.</td>
<td>—</td>
</tr>
<tr>
<td>BOx,y</td>
<td>El adversario tiene entre x e y blots en el outfield.</td>
<td>—</td>
</tr>
<tr>
<td>bj&gt;x</td>
<td>El jugador tiene al menos x blots en el jan.</td>
<td>—</td>
</tr>
<tr>
<td>bj&lt;x</td>
<td>El jugador tiene como máximo x blots en el jan.</td>
<td>—</td>
</tr>
<tr>
<td>bjx,y</td>
<td>El jugador tiene entre x e y blots en el jan.</td>
<td>—</td>
</tr>
<tr>
<td>BJ&gt;x</td>
<td>El adversario tiene al menos x blots en el jan.</td>
<td>—</td>
</tr>
<tr>
<td>BJ&lt;x</td>
<td>El adversario tiene como máximo x blots en el jan.</td>
<td>—</td>
</tr>
<tr>
<td>BJx,y</td>
<td>El adversario tiene entre x e y blots en el jan.</td>
<td>—</td>
</tr>
<tr>
<td><code>t'palabra1;palabra2;...'</code></td>
<td>Los comentarios de la posición contienen al menos una de las palabras.</td>
<td>—</td>
</tr>
<tr>
<td>co</td>
<td>La posición tiene un comentario, sea cual sea su contenido.</td>
<td><code>--has-comment</code></td>
</tr>
<tr>
<td>xco</td>
<td>La posición no tiene ningún comentario.</td>
<td><code>--no-comment</code></td>
</tr>
<tr>
<td>co:user</td>
<td>La posición lleva un comentario de un origen dado: <code>user</code> (escrito por usted), <code>xg</code>, <code>gnubg</code>, <code>bgf</code> (traído por la importación de una partida) o <code>unknown</code>. Repetible (<code>co:xg co:gnubg</code>).</td>
<td><code>--comment-origin</code></td>
</tr>
<tr>
<td><code>m'patrón1,patrón2,...'</code></td>
<td>Las mejores jugadas de fichas que contienen al menos uno de los patrones.</td>
<td>—</td>
</tr>
<tr>
<td><code>m'ND,DT,DP,...'</code></td>
<td>Las mejores decisiones de cubo de No Double/Take, Double Take, Double Pass.</td>
<td>—</td>
</tr>
<tr>
<td>T&gt;x</td>
<td>Fecha de adición de la posición posterior a x (AAAA/MM/DD).</td>
<td>—</td>
</tr>
<tr>
<td>T&lt;x</td>
<td>Fecha de adición de la posición anterior a x (AAAA/MM/DD).</td>
<td>—</td>
</tr>
<tr>
<td>Tx,y</td>
<td>Fecha de adición de la posición entre x e y (AAAA/MM/DD).</td>
<td>—</td>
</tr>
<tr>
<td>max</td>
<td>Buscar en el match con identificador x (p. ej.: ma3).</td>
<td><code>--match-ids</code></td>
</tr>
<tr>
<td>max,y</td>
<td>Buscar en los matches con identificadores de x a y (p. ej.: ma2,5).</td>
<td><code>--match-ids</code></td>
</tr>
<tr>
<td>tnx</td>
<td>Buscar en el torneo con identificador x (p. ej.: tn1).</td>
<td><code>--tournament-ids</code></td>
</tr>
<tr>
<td>tnx,y</td>
<td>Buscar en los torneos con identificadores de x a y (p. ej.: tn1,3).</td>
<td><code>--tournament-ids</code></td>
</tr>
<tr>
<td>idx</td>
<td>Buscar la posición con identificador x (p. ej. id12).</td>
<td><code>--position-ids</code></td>
</tr>
<tr>
<td>idx,y</td>
<td>Buscar las posiciones con identificadores de x a y (p. ej. id5,10).</td>
<td><code>--position-ids</code></td>
</tr>
<tr>
<td><code>pl'nombre'</code></td>
<td>Buscar posiciones de una partida en la que participó el jugador indicado, en cualquier lado (ej: <code>pl'Alice'</code>). No distingue mayúsculas y minúsculas.</td>
<td>—</td>
</tr>
</tbody>
</table>
<h3>Comandos diversos</h3>
<table>
<thead>
<tr>
<th>Comando</th>
<th>Acción</th>
</tr>
</thead>
<tbody>
<tr>
<td>clear, cl</td>
<td>Borra el historial de comandos.</td>
</tr>
</tbody>
</table>
`,
    about: `
<h3>Versión</h3>
<p>Versión de la aplicación: {appVersion}</p>
<p>Versión de la base de datos: {dbVersion}</p>
<p>
    <a href="https://kevung.github.io/blunderDB/es/" target="_blank" rel="noopener noreferrer">Documentación en línea</a> ·
    <a href="https://kevung.github.io/blunderDB/es/historique.html" target="_blank" rel="noopener noreferrer">Historial de versiones</a>
</p>

<h3>Autor</h3>
<p><strong>Kévin Unger &lt;blunderdb@proton.me&gt;</strong></p>
<p>También puedes encontrarme en Heroes con el apodo <strong>postmanpat</strong>.</p>
<p>
    Desarrollé blunderDB inicialmente para mi uso personal, para detectar patrones en mis errores. Pero es muy grato recibir comentarios, sobre todo cuando se han dedicado muchas horas al diseño, la
    programación, la depuración... Así que no dudes en escribirme para compartir tus comentarios.
</p>
<p>Aquí tienes varias formas de contactarme:</p>
<ul>
    <li>Únete al servidor de Discord de blunderDB: <a href="https://discord.gg/DA5PpzM9En" target="_blank" rel="noopener noreferrer">discord.gg/DA5PpzM9En</a>,</li>
    <li>Habla conmigo si coincidimos en un torneo,</li>
    <li>Envíame un correo electrónico,</li>
</ul>
<h3>Licencia</h3>
<p>
    blunderDB se distribuye bajo la licencia MIT. Esto significa que eres libre de usar, copiar, modificar, fusionar, publicar, distribuir, sublicenciar y/o vender copias del software, siempre que el
    aviso de copyright original y este aviso de permiso se incluyan en todas las copias o partes sustanciales del software.
</p>
<h3>Agradecimientos</h3>
<p>Dedico este pequeño software a mi pareja <strong>Anne-Claire</strong> y a nuestra querida hija <strong>Perrine</strong>. Quiero dar las gracias especialmente a algunos amigos:</p>
<ul>
    <li>
        <strong>Tristan Remille</strong>, por iniciarme en el backgammon con alegría y amabilidad; por mostrarme el Camino para comprender este maravilloso juego; por seguir apoyándome a pesar de mis
        pobres intentos de jugar mejor.
    </li>
    <li><strong>Nicolas Harmand</strong>, un compañero alegre durante más de una década en grandes aventuras, y un fantástico compañero de juego desde que le picó el gusanillo del backgammon.</li>
</ul>
<h3>Créditos</h3>
<p>blunderDB incorpora código, datos y fuentes de otras personas. Lo esencial:</p>
<ul>
    <li>
        La red neuronal <strong>strehl-prob5-512-512-256-128</strong> es obra de <strong>Alexander Strehl</strong> (<em>alexstrehl/backgammon-ai-engine</em>, MIT). La búsqueda, el modelo de cubo y la
        tabla de match equity que la rodean son la configuración propia de <strong>gammonNet</strong> (<a href="https://github.com/kevung/gammonNet" target="_blank" rel="noopener noreferrer"
            >github.com/kevung/gammonNet</a
        >, MIT).
    </li>
    <li>La tabla de match equity Kazaross-XG2 (MET) es obra de <strong>Neil Kazaross</strong>.</li>
    <li>Las tablas de puntos de aceptación y de valores de gammon están tomadas del libro <em>The Theory of Backgammon</em> de <strong>Dirk Schiemann</strong>.</li>
    <li>
        Las bases de datos de bear off de un solo lado (6 puntos, 15 fichas, para el EPC) y de dos lados (6 puntos, 6 fichas, para los veredictos de cubo en carreras) se generaron con
        <strong>GNU Backgammon</strong> (GNUbg). GNUbg es software libre bajo la GPL; estas tablas son datos producidos por él, acreditados como tales.
    </li>
    <li>Los archivos de match se leen con <em>xgparser</em>, <em>gnubgparser</em> y <em>bgfparser</em> (MIT).</li>
    <li>Del lado de Go: <em>modernc.org/sqlite</em> (BSD-3-Clause), <em>pgx</em>, <em>Wails</em> y <em>go-fsrs</em> (MIT).</li>
    <li>Del lado de la interfaz: <em>Svelte</em>, <em>two.js</em>, <em>Chart.js</em> y <em>driver.js</em> (MIT).</li>
    <li>Las fuentes <em>Nunito</em> y <em>Noto Sans JP</em> (SIL Open Font License 1.1).</li>
</ul>
<p>
    El inventario completo, con el texto de las licencias, es el archivo <strong>THIRD_PARTY.md</strong> que se distribuye con blunderDB (<a
        href="https://github.com/kevung/blunderDB/blob/main/THIRD_PARTY.md"
        target="_blank"
        rel="noopener noreferrer"
        >github.com/kevung/blunderDB</a
    >).
</p>
`
};
