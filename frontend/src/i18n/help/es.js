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
<h3>Introducción</h3>
<p>
    blunderDB es un software para crear bases de datos de posiciones de backgammon. Su principal fortaleza es ofrecer un único lugar donde reunir las posiciones que un jugador ha encontrado (en línea,
    en torneos) y poder volver a estudiar estas posiciones filtrándolas según diversos filtros combinables arbitrariamente. blunderDB también puede usarse para crear catálogos de posiciones de
    referencia.
</p>
<p>Las posiciones se almacenan en una base de datos representada por un archivo .db.</p>

<h3>Interacciones principales</h3>
<p>Las principales interacciones posibles con blunderDB son:</p>
<ul>
    <li>añadir una nueva posición,</li>
    <li>modificar una posición existente,</li>
    <li>copiar el tablero como imagen PNG al portapapeles (<strong>Ctrl+X</strong>), o el tablero con su análisis (<strong>Ctrl+X, Ctrl+X</strong>),</li>
    <li>eliminar una posición existente,</li>
    <li>buscar una o varias posiciones,</li>
    <li>importar matches de diversas fuentes (XG, GNUbg, BGBlitz, Jellyfish), incluyendo los comentarios de los archivos XG,</li>
    <li>recorrer las jugadas de un match importado,</li>
    <li>organizar las posiciones en colecciones,</li>
    <li>organizar los matches en torneos,</li>
    <li>analizar en lote, desde una terminal, las posiciones sin análisis gracias al evaluador gammonNet integrado (comando <strong>analyze</strong> de blunderDB).</li>
</ul>
<p>El usuario puede etiquetar libremente las posiciones y anotarlas con comentarios.</p>

<h3>Descripción de la interfaz</h3>
<p>La interfaz de blunderDB está estructurada de arriba a abajo de la siguiente manera:</p>
<ul>
    <li>[en la parte superior] la barra de herramientas, que reúne todas las operaciones principales que se pueden realizar sobre la base de datos,</li>
    <li>[en el centro] el área de visualización principal, que permite mostrar o editar posiciones de backgammon,</li>
    <li>[en la parte inferior] la barra de estado, que integra la línea de comandos y presenta diversa información sobre la posición actual.</li>
</ul>
<p>Se pueden mostrar paneles para:</p>
<ul>
    <li>mostrar los datos de análisis asociados a la posición actual (de XG, GNUbg o BGBlitz),</li>
    <li>mostrar, añadir o modificar comentarios,</li>
    <li>recorrer los matches importados y navegar por sus jugadas (panel Match),</li>
    <li>gestionar colecciones de posiciones (panel Colección),</li>
    <li>estudiar posiciones con repetición espaciada (panel Anki),</li>
    <li>gestionar torneos (panel Torneo),</li>
    <li>mostrar estadísticas de rendimiento (panel Stats),</li>
    <li>calcular valores de EPC para posiciones de bear off (panel Eval),</li>
    <li>consultar filtros de búsqueda guardados (panel Biblioteca de filtros),</li>
    <li>consultar el historial de búsquedas (panel Historial de búsquedas).</li>
</ul>
<p>El área de visualización principal ofrece al usuario:</p>
<ul>
    <li>un tablero para mostrar o editar una posición de backgammon,</li>
    <li>el nivel y el propietario del cubo,</li>
    <li>el pip count de cada jugador,</li>
    <li>el marcador de cada jugador,</li>
    <li>los dados a jugar. Si no se muestra ningún valor en los dados, la posición de los dados indica qué jugador tiene el turno y que la posición es una decisión de cubo.</li>
</ul>
<p>La barra de estado muestra de izquierda a derecha:</p>
<ul>
    <li>la línea de comandos (pulsa <strong>Espacio</strong> para abrirla),</li>
    <li>un mensaje informativo relacionado con la última operación realizada,</li>
    <li>el índice de la posición actual, seguido del número total de posiciones (o información de jugada/partida al navegar por un match).</li>
</ul>
<p>En el caso de posiciones resultantes de una búsqueda del usuario, el número de posiciones indicado en la barra de estado corresponde al número de posiciones filtradas.</p>

<h3>Navegar por las posiciones</h3>
<p>Por defecto, blunderDB te permite:</p>
<ul>
    <li>desplazarte por las distintas posiciones de la biblioteca actual,</li>
    <li>mostrar la información de análisis asociada a una posición,</li>
    <li>mostrar, añadir y modificar comentarios sobre una posición.</li>
</ul>

<h3>Editar posiciones</h3>
<p>
    Pulsar la tecla <strong>Tab</strong> abre el panel de búsqueda y permite editar una posición en el tablero para añadirla a la base de datos o para definir una estructura de posición que buscar. La
    distribución de las fichas, el cubo, el marcador y el turno pueden modificarse con el ratón.
</p>

<h3>Línea de comandos</h3>
<p>
    La línea de comandos, integrada en la barra de estado, permite realizar todas las funcionalidades de blunderDB: operaciones sobre la base de datos, navegación por las posiciones, mostrar análisis
    y comentarios, buscar posiciones con filtros... Tras familiarizarte con la interfaz, se recomienda usar progresivamente la línea de comandos, que permite un uso potente y fluido de blunderDB,
    especialmente para las funcionalidades de búsqueda de posiciones.
</p>
<p>
    Para abrir la línea de comandos, pulsa la tecla <strong>Espacio</strong>. Aparece un prompt en la barra de estado. Escribe tu comando y pulsa <strong>Enter</strong> para ejecutarlo. Pulsa
    <strong>Escape</strong>
    para cancelar.
</p>
<p>
    blunderDB ejecuta las consultas enviadas por el usuario siempre que sean válidas y modifica inmediatamente el estado de la base de datos si es necesario. No se requieren acciones de guardado
    explícitas por parte del usuario.
</p>
<p>
    Para refinar una búsqueda dentro de posiciones previamente filtradas, usa el comando <strong>ss</strong> seguido de filtros (p. ej., <strong>ss nc</strong>). Esto restringe la búsqueda a solo las
    posiciones actualmente mostradas, lo que permite ir acotando progresivamente los resultados. El panel de búsqueda (<strong>Ctrl+F</strong>) también ofrece una casilla "Buscar en los resultados
    actuales" para la misma funcionalidad.
</p>

<h3>Panel Eval</h3>
<p>
    El panel <strong>Eval</strong> evalúa cualquier posición que esté en el tablero: probabilidades de victoria, gammon y backgammon, equity, jugadas candidatas ordenadas y la única decisión que la
    posición exige — jugar una tirada o doblar. El cálculo lo hace gammonNet, integrado: no hacen falta ni eXtreme Gammon ni GNU Backgammon.
</p>
<p>
    Para abrirlo, pulse <strong>Ctrl+E</strong>, haga clic en la pestaña Eval del panel inferior o escriba <strong>epc</strong> en la línea de comandos. El tablero se abre con una configuración de
    bearoff estándar (15 fichas), salvo que se le haya enviado una posición de la base. Las fichas se añaden y se quitan libremente con el ratón; la evaluación sigue cada cambio.
</p>
<p>
    En una posición de bearoff el panel <strong>se especializa</strong>: una segunda tabla, por jugador, muestra el EPC (Effective Pip Count) calculado a partir de la base de bearoff unilateral de 6
    puntos de GNUbg —
</p>
<ul>
    <li><strong>EPC</strong>: el número medio de pips necesarios para sacar todas las fichas,</li>
    <li><strong>Pip Count</strong>: el pip count bruto,</li>
    <li><strong>Wastage</strong>: la diferencia entre el EPC y el pip count,</li>
    <li><strong>Avg Rolls</strong>: el número medio de tiradas para sacar todas las fichas,</li>
    <li><strong>Std Dev</strong>: la desviación típica de ese número de tiradas.</li>
</ul>
<p>Cuando ambos jugadores tienen fichas en su casa, una sección de comparación muestra las diferencias de EPC y de pip count.</p>
<p>
    En una carrera pura, otra tabla muestra las probabilidades de victoria de ambos jugadores y, cuando la posición está cubierta por una base two-sided (la integrada hasta 6 fichas por jugador, la
    ampliada descargable hasta 11 desde la pestaña Bearoff de la configuración), los equities money exactos y la mejor decisión de cubo. Fuera de ese dominio la probabilidad de victoria se estima
    (distintivo «estimado» con su margen de error) y no se muestra ninguna decisión. El jugador en turno se cambia haciendo clic en el rectángulo de salida/marcador de un jugador, y la posición del
    cubo haciendo clic en el cubo del tablero.
</p>
<p>
    La casilla <strong>Desafío</strong> oculta los resultados cada vez que se modifica la posición; haga clic en una zona para revelarla — ideal para practicar un equity, un EPC o una decisión de cubo
    antes de comprobar.
</p>
<p>Para cerrar el panel Eval, pulse <strong>Ctrl+E</strong> de nuevo o cambie de pestaña.</p>

<h3>Navegación por matches</h3>
<p>
    blunderDB permite recorrer las jugadas de los matches importados. Abre el panel Match con <strong>Ctrl+Tab</strong> y haz doble clic en un match (o pulsa <strong>Enter</strong>) para cargar sus
    posiciones.
</p>
<p>
    Al navegar por un match, la última posición visitada se guarda y se restaura automáticamente. Usa las teclas <strong>Izquierda</strong>/<strong>Derecha</strong> para moverte entre posiciones, y
    <strong>PageUp</strong>/<strong>PageDown</strong> para saltar entre partidas.
</p>
<p>
    El panel de análisis (<strong>Ctrl+L</strong>) muestra el análisis de cada jugada, resaltando la jugada realizada. Pulsa <strong>d</strong> para alternar entre el análisis de fichas y el de cubo.
</p>

<h3>Colecciones</h3>
<p>
    Las colecciones permiten organizar las posiciones en grupos personalizados. Abre el panel Colección con <strong>Ctrl+B</strong> y luego haz doble clic en una colección para recorrer sus
    posiciones. Las colecciones y las posiciones que contienen pueden reordenarse arrastrando y soltando.
</p>

<h3>Anki (repetición espaciada)</h3>
<p>El panel Anki (<strong>Ctrl+K</strong>) ofrece repetición espaciada para estudiar posiciones de backgammon usando el algoritmo FSRS.</p>
<p>
    <strong>Crear mazos:</strong> Haz clic en <em>Nuevo mazo</em> para crear un mazo a partir de una colección o de los resultados de búsqueda actuales. Los mazos basados en búsquedas se sincronizan
    automáticamente cuando se activa la pestaña Anki.
</p>
<p>
    <strong>Repasar:</strong> Selecciona un mazo y haz clic en <em>Estudiar</em> (o haz doble clic en un mazo) para empezar a repasar las tarjetas pendientes. Cada tarjeta muestra la posición
    correspondiente en el tablero. Califica tu recuerdo con las teclas <strong>1</strong> (Otra vez), <strong>2</strong> (Difícil), <strong>3</strong> (Bien) o <strong>4</strong> (Fácil). Pulsa
    <strong>Esc</strong> para detenerte y volver a la lista de mazos.
</p>
<p>
    <strong>Limitar la sesión:</strong> En los ajustes del mazo puede acotar una sesión a un número de tarjetas. La sesión se detiene indicándolo, y el repaso libre sigue disponible para continuar sin
    tocar la planificación. Un límite de <em>0</em> no sirve ninguna tarjeta, lo que no equivale a no tener límite.
</p>
<p>
    <strong>Retención:</strong> La retención objetivo es su elección sobre el compromiso carga/calidad. Los ajustes muestran al lado la retención <em>medida</em> en sus repasos: información, nunca un
    control. Cambiar el objetivo no es retroactivo: cada tarjeta adopta el nuevo ritmo en su próximo repaso.
</p>
<p>
    <strong>Mostrar la respuesta:</strong> La tarjeta plantea una pregunta; piénselo y pulse <strong>Espacio</strong> (o haga clic en la zona oculta) para revelar el análisis registrado de la
    posición. Aparece debajo de los botones de valoración, que siguen al alcance. No es necesario revelarla para valorar, y vuelve a ocultarse en la siguiente tarjeta, no al cambiar de pestaña.
</p>
<p>
    <strong>Detener/Reanudar:</strong> Puedes detener una sesión de repaso en cualquier momento pulsando <strong>Esc</strong>. El botón cambia a <em>Reanudar</em> mostrando tu progreso. Haz clic en él
    para continuar donde lo dejaste.
</p>
<p>
    <strong>Gestión de mazos:</strong> Usa los botones de acción para renombrar, sincronizar, reiniciar o eliminar mazos. Los parámetros de FSRS (retención objetivo, intervalo máximo, fuzz) pueden
    configurarse por mazo en Ajustes (icono de engranaje).
</p>

<h3>Torneos</h3>
<p>Los torneos permiten agrupar matches por evento. Abre el panel Torneo con <strong>Ctrl+Y</strong> para gestionar torneos y asignarles matches.</p>

<h3>Stats</h3>
<p>
    El panel Stats (<strong>Ctrl+D</strong>) muestra estadísticas de rendimiento (PR y coste en MWC) calculadas a partir de todas las posiciones importadas. Usa la barra de filtros para restringir el
    análisis por jugador, torneo, rango de fechas, tipo de decisión o duración del match. Haz clic en cualquier indicador para desglosar las posiciones correspondientes. La pestaña
    <strong>Jugadores</strong> muestra, por jugador, el número de partidas, el balance, las decisiones, el PR (fichas y cubo), el Snowie, los blunders y la suerte medida sobre las tiradas conocidas.
</p>

<h3>Marca y exportación protegida</h3>
<p>Al exportar (<strong>export_db</strong> o el cuadro de diálogo Exportar), se pueden activar libremente dos protecciones independientes, una, la otra, o ambas a la vez:</p>
<ul>
    <li>
        <strong>Marca:</strong> marca el archivo exportado con su origen (quién lo produjo, una nota opcional). La marca está firmada con su identidad del emisor: no puede alterarse ni falsificarse en
        nombre de otra persona — pero no es ineliminable y no impide ninguna copia.
    </li>
    <li>
        <strong>Contraseña:</strong> coloca la exportación en un contenedor cifrado <strong>.dbx</strong>. Protege el archivo durante el transporte, no la base de datos en sí — quien reciba la
        contraseña podrá abrirlo — y el origen sigue siendo legible sin ella.
    </li>
</ul>
<p>
    Su identidad del emisor, la clave que firma sus marcas, se crea automáticamente en la primera exportación marcada con su origen. Consúltela, expórtela o regenérela desde la pestaña
    <strong>Identidad del emisor</strong> de la configuración.
</p>
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
<td>CTRL-SHIFT-I</td>
<td>Importar una base de datos.</td>
</tr>
<tr>
<td>CTRL-SHIFT-S</td>
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
<td>CTRL-SHIFT-F</td>
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
<td>Abre el panel Eval (Effective Pip Count, probabilidad de victoria y veredicto de cubo en bearoff).</td>
</tr>
<tr>
<td>met</td>
<td>Abre la tabla de equidad de match Kazaross-XG2.</td>
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
<td>Importa una o varias posiciones/matches desde un archivo (xg, xgp, sgf, mat, txt, bgf).</td>
</tr>
<tr>
<td>delete, del, d</td>
<td>Elimina la posición actual (se pide confirmación).</td>
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
<table>
<thead>
<tr>
<th>Consulta</th>
<th>Acción</th>
</tr>
</thead>
<tbody>
<tr>
<td>cube, cub, cu, c</td>
<td>La posición cumple la configuración del cubo.</td>
</tr>
<tr>
<td>score, sco, sc, s</td>
<td>La posición cumple el marcador.</td>
</tr>
<tr>
<td>d</td>
<td>La posición cumple el tipo de decisión (ficha o cubo).</td>
</tr>
<tr>
<td>D</td>
<td>La posición cumple la tirada de dados (ambos dados, sin importar el orden).</td>
</tr>
<tr>
<td>D1</td>
<td>La posición cumple la tirada de dados únicamente en el primer dado (el valor del primer dado aparece en cualquiera de los dos dados de la posición).</td>
</tr>
<tr>
<td>xD65</td>
<td>La posición <strong>no</strong> se jugó con la tirada 6-5 (sin importar el orden). El valor se indica en el token; repetible para excluir varias tiradas (<code>xD65 xD54</code>).</td>
</tr>
<tr>
<td>nc</td>
<td>La posición es sin contacto.</td>
</tr>
<tr>
<td>M</td>
<td>La posición o su réplica especular cumple los filtros.</td>
</tr>
<tr>
<td>i</td>
<td>La posición se importó por separado, y no la trajo la importación de una partida.</td>
</tr>
<tr>
<td>fl</td>
<td>La posición fue marcada en el programa de origen, al importar una partida de eXtreme Gammon.</td>
</tr>
<tr>
<td>x</td>
<td>La posición no contiene ninguna ficha de la estructura de exclusión (pestaña « Except » del panel de búsqueda).</td>
</tr>
<tr>
<td>p&gt;x</td>
<td>El jugador tiene al menos x pips de desventaja en la carrera.</td>
</tr>
<tr>
<td>p&lt;x</td>
<td>El jugador tiene como máximo x pips de desventaja en la carrera.</td>
</tr>
<tr>
<td>px,y</td>
<td>El jugador tiene entre x e y pips de desventaja en la carrera.</td>
</tr>
<tr>
<td>P&gt;x</td>
<td>El jugador tiene una carrera de al menos x pips.</td>
</tr>
<tr>
<td>P&lt;x</td>
<td>El jugador tiene una carrera de como máximo x pips.</td>
</tr>
<tr>
<td>Px,y</td>
<td>El jugador tiene una carrera entre x e y pips.</td>
</tr>
<tr>
<td>e&gt;x</td>
<td>La equidad (en milipuntos) de la posición es mayor que x.</td>
</tr>
<tr>
<td>e&lt;x</td>
<td>La equidad (en milipuntos) de la posición es menor que x.</td>
</tr>
<tr>
<td>ex,y</td>
<td>La equidad (en milipuntos) de la posición está comprendida entre x e y.</td>
</tr>
<tr>
<td>E&gt;x</td>
<td>El error de la jugada realizada por el jugador 1 (en milipuntos) es mayor que x.</td>
</tr>
<tr>
<td>E&lt;x</td>
<td>El error de la jugada realizada por el jugador 1 (en milipuntos) es menor que x.</td>
</tr>
<tr>
<td>Ex,y</td>
<td>El error de la jugada realizada por el jugador 1 (en milipuntos) está comprendido entre x e y.</td>
</tr>
<tr>
<td>w&gt;x</td>
<td>El jugador tiene probabilidades de victoria superiores al x %.</td>
</tr>
<tr>
<td>w&lt;x</td>
<td>El jugador tiene probabilidades de victoria inferiores al x %.</td>
</tr>
<tr>
<td>wx,y</td>
<td>El jugador tiene probabilidades de victoria entre el x % y el y %.</td>
</tr>
<tr>
<td>g&gt;x</td>
<td>El jugador tiene probabilidades de gammon superiores al x %.</td>
</tr>
<tr>
<td>g&lt;x</td>
<td>El jugador tiene probabilidades de gammon inferiores al x %.</td>
</tr>
<tr>
<td>gx,y</td>
<td>El jugador tiene probabilidades de gammon entre el x % y el y %.</td>
</tr>
<tr>
<td>b&gt;x</td>
<td>El jugador tiene probabilidades de backgammon superiores al x %.</td>
</tr>
<tr>
<td>b&lt;x</td>
<td>El jugador tiene probabilidades de backgammon inferiores al x %.</td>
</tr>
<tr>
<td>bx,y</td>
<td>El jugador tiene probabilidades de backgammon entre el x % y el y %.</td>
</tr>
<tr>
<td>W&gt;x</td>
<td>El adversario tiene probabilidades de victoria superiores al x %.</td>
</tr>
<tr>
<td>W&lt;x</td>
<td>El adversario tiene probabilidades de victoria inferiores al x %.</td>
</tr>
<tr>
<td>Wx,y</td>
<td>El adversario tiene probabilidades de victoria entre el x % y el y %.</td>
</tr>
<tr>
<td>G&gt;x</td>
<td>El adversario tiene probabilidades de gammon superiores al x %.</td>
</tr>
<tr>
<td>G&lt;x</td>
<td>El adversario tiene probabilidades de gammon inferiores al x %.</td>
</tr>
<tr>
<td>Gx,y</td>
<td>El adversario tiene probabilidades de gammon entre el x % y el y %.</td>
</tr>
<tr>
<td>B&gt;x</td>
<td>El adversario tiene probabilidades de backgammon superiores al x %.</td>
</tr>
<tr>
<td>B&lt;x</td>
<td>El adversario tiene probabilidades de backgammon inferiores al x %.</td>
</tr>
<tr>
<td>Bx,y</td>
<td>El adversario tiene probabilidades de backgammon entre el x % y el y %.</td>
</tr>
<tr>
<td>o&gt;x</td>
<td>El jugador tiene al menos x fichas retiradas.</td>
</tr>
<tr>
<td>o&lt;x</td>
<td>El jugador tiene como máximo x fichas retiradas.</td>
</tr>
<tr>
<td>ox,y</td>
<td>El jugador tiene entre x e y fichas retiradas.</td>
</tr>
<tr>
<td>O&gt;x</td>
<td>El adversario tiene al menos x fichas retiradas.</td>
</tr>
<tr>
<td>O&lt;x</td>
<td>El adversario tiene como máximo x fichas retiradas.</td>
</tr>
<tr>
<td>Ox,y</td>
<td>El adversario tiene entre x e y fichas retiradas.</td>
</tr>
<tr>
<td>k&gt;x</td>
<td>El jugador tiene al menos x fichas rezagadas.</td>
</tr>
<tr>
<td>k&lt;x</td>
<td>El jugador tiene como máximo x fichas rezagadas.</td>
</tr>
<tr>
<td>kx,y</td>
<td>El jugador tiene entre x e y fichas rezagadas.</td>
</tr>
<tr>
<td>K&gt;x</td>
<td>El adversario tiene al menos x fichas rezagadas.</td>
</tr>
<tr>
<td>K&lt;x</td>
<td>El adversario tiene como máximo x fichas rezagadas.</td>
</tr>
<tr>
<td>Kx,y</td>
<td>El adversario tiene entre x e y fichas rezagadas.</td>
</tr>
<tr>
<td>z&gt;x</td>
<td>El jugador tiene al menos x fichas en la zona.</td>
</tr>
<tr>
<td>z&lt;x</td>
<td>El jugador tiene como máximo x fichas en la zona.</td>
</tr>
<tr>
<td>zx,y</td>
<td>El jugador tiene entre x e y fichas en la zona.</td>
</tr>
<tr>
<td>Z&gt;x</td>
<td>El adversario tiene al menos x fichas en la zona.</td>
</tr>
<tr>
<td>Z&lt;x</td>
<td>El adversario tiene como máximo x fichas en la zona.</td>
</tr>
<tr>
<td>Zx,y</td>
<td>El adversario tiene entre x e y fichas en la zona.</td>
</tr>
<tr>
<td>bo&gt;x</td>
<td>El jugador tiene al menos x blots en el outfield.</td>
</tr>
<tr>
<td>bo&lt;x</td>
<td>El jugador tiene como máximo x blots en el outfield.</td>
</tr>
<tr>
<td>box,y</td>
<td>El jugador tiene entre x e y blots en el outfield.</td>
</tr>
<tr>
<td>BO&gt;x</td>
<td>El adversario tiene al menos x blots en el outfield.</td>
</tr>
<tr>
<td>BO&lt;x</td>
<td>El adversario tiene como máximo x blots en el outfield.</td>
</tr>
<tr>
<td>BOx,y</td>
<td>El adversario tiene entre x e y blots en el outfield.</td>
</tr>
<tr>
<td>bj&gt;x</td>
<td>El jugador tiene al menos x blots en el jan.</td>
</tr>
<tr>
<td>bj&lt;x</td>
<td>El jugador tiene como máximo x blots en el jan.</td>
</tr>
<tr>
<td>bjx,y</td>
<td>El jugador tiene entre x e y blots en el jan.</td>
</tr>
<tr>
<td>BJ&gt;x</td>
<td>El adversario tiene al menos x blots en el jan.</td>
</tr>
<tr>
<td>BJ&lt;x</td>
<td>El adversario tiene como máximo x blots en el jan.</td>
</tr>
<tr>
<td>BJx,y</td>
<td>El adversario tiene entre x e y blots en el jan.</td>
</tr>
<tr>
<td>t'palabra1;palabra2;...'</td>
<td>Los comentarios de la posición contienen al menos una de las palabras.</td>
</tr>
<tr>
<td>co</td>
<td>La posición tiene un comentario, sea cual sea su contenido.</td>
</tr>
<tr>
<td>xco</td>
<td>La posición no tiene ningún comentario.</td>
</tr>
<tr>
<td>m'patrón1,patrón2,...'</td>
<td>Las mejores jugadas de fichas que contienen al menos uno de los patrones.</td>
</tr>
<tr>
<td>m'ND,DT,DP,...'</td>
<td>Las mejores decisiones de cubo de No Double/Take, Double Take, Double Pass.</td>
</tr>
<tr>
<td>T&gt;x</td>
<td>Fecha de adición de la posición posterior a x (AAAA/MM/DD).</td>
</tr>
<tr>
<td>T&lt;x</td>
<td>Fecha de adición de la posición anterior a x (AAAA/MM/DD).</td>
</tr>
<tr>
<td>Tx,y</td>
<td>Fecha de adición de la posición entre x e y (AAAA/MM/DD).</td>
</tr>
<tr>
<td>max</td>
<td>Buscar en el match con identificador x (p. ej.: ma3).</td>
</tr>
<tr>
<td>max,y</td>
<td>Buscar en los matches con identificadores de x a y (p. ej.: ma2,5).</td>
</tr>
<tr>
<td>tnx</td>
<td>Buscar en el torneo con identificador x (p. ej.: tn1).</td>
</tr>
<tr>
<td>tnx,y</td>
<td>Buscar en los torneos con identificadores de x a y (p. ej.: tn1,3).</td>
</tr>
<tr>
<td>idx</td>
<td>Buscar la posición con identificador x (p. ej. id12).</td>
</tr>
<tr>
<td>idx,y</td>
<td>Buscar las posiciones con identificadores de x a y (p. ej. id5,10).</td>
</tr>
<tr>
<td>pl'nombre'</td>
<td>Buscar posiciones de una partida en la que participó el jugador indicado, en cualquier lado (p. ej. pl'Alice'). No distingue mayúsculas y minúsculas.</td>
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
