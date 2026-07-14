// Spanish help content (translated from en.js).
// Each value is the verbatim inner HTML of the corresponding HelpModal tab.
export default {
    manual: `
                    <h3>Introducción</h3>
                    <p>
                        blunderDB es un software para crear bases de datos de posiciones de backgammon. Su principal fortaleza es ofrecer un único lugar donde reunir las posiciones que un jugador ha encontrado (en línea,
                        en torneos) y poder volver a estudiar estas posiciones filtrándolas según diversos filtros combinables arbitrariamente. blunderDB también puede usarse para crear catálogos
                        de posiciones de referencia.
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
                        <li>organizar los matches en torneos.</li>
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
                        <li>calcular valores de EPC para posiciones de bear off (panel EPC),</li>
                        <li>consultar filtros de búsqueda guardados (panel Biblioteca de filtros),</li>
                        <li>consultar el historial de búsquedas (panel Historial de búsquedas),</li>
                        <li>ver los registros de operaciones (panel Log).</li>
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
                        Pulsar la tecla <strong>Tab</strong> abre el panel de búsqueda y permite editar una posición en el tablero para añadirla a la base de datos o para definir una estructura de posición que buscar.
                        La distribución de las fichas, el cubo, el marcador y el turno pueden modificarse con el ratón.
                    </p>

                    <h3>Línea de comandos</h3>
                    <p>
                        La línea de comandos, integrada en la barra de estado, permite realizar todas las funcionalidades de blunderDB: operaciones sobre la base de datos, navegación por las posiciones, mostrar análisis y
                        comentarios, buscar posiciones con filtros... Tras familiarizarte con la interfaz, se recomienda usar progresivamente la línea de comandos, que permite un uso potente y
                        fluido de blunderDB, especialmente para las funcionalidades de búsqueda de posiciones.
                    </p>
                    <p>
                        Para abrir la línea de comandos, pulsa la tecla <strong>Espacio</strong>. Aparece un prompt en la barra de estado. Escribe tu comando y pulsa <strong>Enter</strong> para ejecutarlo. Pulsa
                        <strong>Escape</strong>
                        para cancelar. El historial de comandos y los resultados se registran en el panel <strong>Log</strong>.
                    </p>
                    <p>
                        blunderDB ejecuta las consultas enviadas por el usuario siempre que sean válidas y modifica inmediatamente el estado de la base de datos si es necesario. No se requieren acciones de guardado explícitas
                        por parte del usuario.
                    </p>
                    <p>
                        Para refinar una búsqueda dentro de posiciones previamente filtradas, usa el comando <strong>ss</strong> seguido de filtros (p. ej., <strong>ss nc</strong>). Esto restringe la búsqueda a
                        solo las posiciones actualmente mostradas, lo que permite ir acotando progresivamente los resultados. El panel de búsqueda (<strong>Ctrl+F</strong>) también ofrece una casilla "Buscar en los resultados actuales"
                        para la misma funcionalidad.
                    </p>

                    <h3>Calculadora EPC</h3>
                    <p>La calculadora EPC (Effective Pip Count) calcula el pip count efectivo de las posiciones de bear off. Utiliza la base de datos de bear off de un solo lado de 6 puntos de GNUbg para obtener valores de EPC exactos.</p>
                    <p>
                        Para abrir el panel EPC, pulsa <strong>Ctrl+E</strong>, haz clic en la pestaña EPC del panel inferior o escribe <strong>epc</strong> en la línea de comandos. El tablero se inicializa con una configuración estándar
                        de bear off (15 fichas).
                    </p>
                    <p>
                        Puedes añadir o quitar libremente fichas en los puntos del cuadro interior con el ratón. Los valores de EPC se muestran en tiempo real en el panel EPC dedicado, indicando para cada jugador:
                    </p>
                    <ul>
                        <li><strong>EPC</strong>: el número medio de pips necesarios para sacar todas las fichas,</li>
                        <li><strong>Pip Count</strong>: el pip count bruto,</li>
                        <li><strong>Wastage</strong>: la diferencia entre el EPC y el pip count,</li>
                        <li><strong>Avg Rolls</strong>: número medio de tiradas para sacar todas las fichas,</li>
                        <li><strong>Std Dev</strong>: desviación estándar del número de tiradas.</li>
                    </ul>
                    <p>Cuando ambos jugadores tienen fichas en su cuadro interior, una sección de comparación muestra las diferencias de EPC y de pip count.</p>
                    <p>Para cerrar el panel EPC, pulsa <strong>Ctrl+E</strong> de nuevo o cambia a otra pestaña.</p>

                    <h3>Navegación por matches</h3>
                    <p>
                        blunderDB permite recorrer las jugadas de los matches importados. Abre el panel Match con <strong>Ctrl+Tab</strong> y haz doble clic en un match (o pulsa <strong>Enter</strong>)
                        para cargar sus posiciones.
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
                        Las colecciones permiten organizar las posiciones en grupos personalizados. Abre el panel Colección con <strong>Ctrl+B</strong> y luego haz doble clic en una colección para recorrer sus posiciones.
                        Las colecciones y las posiciones que contienen pueden reordenarse arrastrando y soltando.
                    </p>

                    <h3>Anki (repetición espaciada)</h3>
                    <p>El panel Anki (<strong>Ctrl+K</strong>) ofrece repetición espaciada para estudiar posiciones de backgammon usando el algoritmo FSRS.</p>
                    <p>
                        <strong>Crear mazos:</strong> Haz clic en <em>Nuevo mazo</em> para crear un mazo a partir de una colección o de los resultados de búsqueda actuales. Los mazos basados en búsquedas se sincronizan automáticamente cuando se activa la pestaña Anki.
                    </p>
                    <p>
                        <strong>Repasar:</strong> Selecciona un mazo y haz clic en <em>Estudiar</em> (o haz doble clic en un mazo) para empezar a repasar las tarjetas pendientes. Cada tarjeta muestra la posición correspondiente en el
                        tablero. Califica tu recuerdo con las teclas <strong>1</strong> (Otra vez), <strong>2</strong> (Difícil), <strong>3</strong> (Bien) o <strong>4</strong> (Fácil). Pulsa <strong>Esc</strong> para detenerte
                        y volver a la lista de mazos.
                    </p>
                    <p>
                        <strong>Detener/Reanudar:</strong> Puedes detener una sesión de repaso en cualquier momento pulsando <strong>Esc</strong>. El botón cambia a <em>Reanudar</em> mostrando tu progreso. Haz clic en él para
                        continuar donde lo dejaste.
                    </p>
                    <p>
                        <strong>Gestión de mazos:</strong> Usa los botones de acción para renombrar, sincronizar, reiniciar o eliminar mazos. Los parámetros de FSRS (retención objetivo, intervalo máximo, fuzz) pueden configurarse por mazo
                        en Ajustes (icono de engranaje).
                    </p>

                    <h3>Torneos</h3>
                    <p>Los torneos permiten agrupar matches por evento. Abre el panel Torneo con <strong>Ctrl+Y</strong> para gestionar torneos y asignarles matches.</p>

                    <h3>Stats</h3>
                    <p>
                        El panel Stats (<strong>Ctrl+D</strong>) muestra estadísticas de rendimiento (PR y coste en MWC) calculadas a partir de todas las posiciones importadas. Usa la barra de filtros para restringir el análisis por
                        jugador, torneo, rango de fechas, tipo de decisión o duración del match. Haz clic en cualquier indicador para desglosar las posiciones correspondientes.
                    </p>
`,
    shortcuts: `
                    <h3>Base de datos</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Atajo</th>
                                <th>Descripción</th>
                            </tr>
                        </thead>

                        <tbody>
                            <tr>
                                <td>Ctrl + N</td>
                                <td>Nueva base de datos</td>
                            </tr>

                            <tr>
                                <td>Ctrl + O</td>
                                <td>Abrir base de datos</td>
                            </tr>

                            <tr>
                                <td>Ctrl + Shift + I</td>
                                <td>Importar base de datos</td>
                            </tr>

                            <tr>
                                <td>Ctrl + Shift + S</td>
                                <td>Exportar base de datos</td>
                            </tr>

                            <tr>
                                <td>Ctrl + Q</td>
                                <td>Salir de blunderDB</td>
                            </tr>

                            <tr>
                                <td>Ctrl + M</td>
                                <td>Editar metadatos</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Posición</h3>

                    <table>
                        <thead>
                            <tr>
                                <th>Atajo</th>
                                <th>Descripción</th>
                            </tr>
                        </thead>

                        <tbody>
                            <tr>
                                <td>Ctrl + I</td>
                                <td>Importar posición o match</td>
                            </tr>

                            <tr>
                                <td>Ctrl + C</td>
                                <td>Copiar posición (también copia al portapapeles del tablero)</td>
                            </tr>

                            <tr>
                                <td>Ctrl + X</td>
                                <td>Copiar imagen del tablero al portapapeles (PNG)</td>
                            </tr>

                            <tr>
                                <td>Ctrl + X, Ctrl + X</td>
                                <td>Copiar imagen del tablero + análisis al portapapeles (PNG)</td>
                            </tr>

                            <tr>
                                <td>Ctrl + V</td>
                                <td>Pegar posición (en el panel de búsqueda: pegar al tablero)</td>
                            </tr>

                            <tr>
                                <td>Ctrl + S</td>
                                <td>Guardar posición</td>
                            </tr>

                            <tr>
                                <td>Ctrl + U</td>
                                <td>Actualizar posición</td>
                            </tr>

                            <tr>
                                <td>Del</td>
                                <td>Eliminar posición</td>
                            </tr>

                            <tr>
                                <td>Backspace</td>
                                <td>Reiniciar tablero, cubo, marcador y dados</td>
                            </tr>

                            <tr>
                                <td>Ctrl + G</td>
                                <td>Mostrar metadatos de la posición</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Navegación</h3>

                    <table>
                        <thead>
                            <tr>
                                <th>Atajo</th>
                                <th>Descripción</th>
                            </tr>
                        </thead>

                        <tbody>
                            <tr>
                                <td>Ctrl + R</td>
                                <td>Recargar todas las posiciones</td>
                            </tr>

                            <tr>
                                <td>PageUp, h</td>
                                <td>Primera posición / Partida anterior (navegación por match)</td>
                            </tr>

                            <tr>
                                <td>Left, k</td>
                                <td>Posición anterior</td>
                            </tr>

                            <tr>
                                <td>Right, j</td>
                                <td>Posición siguiente</td>
                            </tr>

                            <tr>
                                <td>PageDown, l</td>
                                <td>Última posición / Partida siguiente (navegación por match)</td>
                            </tr>

                            <tr>
                                <td>r</td>
                                <td>Cargar posición aleatoria</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Visualización</h3>

                    <table>
                        <thead>
                            <tr>
                                <th>Atajo</th>
                                <th>Descripción</th>
                            </tr>
                        </thead>

                        <tbody>
                            <tr>
                                <td>Ctrl + ArrowLeft</td>
                                <td>Orientar el tablero a la izquierda</td>
                            </tr>

                            <tr>
                                <td>Ctrl + ArrowRight</td>
                                <td>Orientar el tablero a la derecha</td>
                            </tr>

                            <tr>
                                <td>p</td>
                                <td>Mostrar/ocultar pipcount</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Acciones</h3>

                    <table>
                        <thead>
                            <tr>
                                <th>Atajo</th>
                                <th>Descripción</th>
                            </tr>
                        </thead>

                        <tbody>
                            <tr>
                                <td>Tab</td>
                                <td>Abrir panel de búsqueda (editor de posición)</td>
                            </tr>

                            <tr>
                                <td>Space</td>
                                <td>Abrir línea de comandos</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Herramientas</h3>

                    <table>
                        <thead>
                            <tr>
                                <th>Atajo</th>
                                <th>Descripción</th>
                            </tr>
                        </thead>

                        <tbody>
                            <tr>
                                <td>Ctrl + L</td>
                                <td>Mostrar análisis</td>
                            </tr>

                            <tr>
                                <td>Ctrl + P</td>
                                <td>Escribir comentarios</td>
                            </tr>

                            <tr>
                                <td>Ctrl + K</td>
                                <td>Mostrar panel Anki</td>
                            </tr>

                            <tr>
                                <td>Ctrl + F</td>
                                <td>Panel de búsqueda</td>
                            </tr>

                            <tr>
                                <td>Ctrl + Tab</td>
                                <td>Panel Match</td>
                            </tr>

                            <tr>
                                <td>Ctrl + B</td>
                                <td>Panel Colección</td>
                            </tr>

                            <tr>
                                <td>Ctrl + Y</td>
                                <td>Panel Torneos</td>
                            </tr>

                            <tr>
                                <td>Ctrl + D</td>
                                <td>Panel Stats</td>
                            </tr>

                            <tr>
                                <td>Ctrl + E</td>
                                <td>Panel EPC</td>
                            </tr>

                            <tr>
                                <td>?</td>
                                <td>Abrir ayuda</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Línea de comandos</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Atajo</th>
                                <th>Descripción</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>Up</td>
                                <td>Recorrer el historial de comandos hacia arriba</td>
                            </tr>
                            <tr>
                                <td>Down</td>
                                <td>Recorrer el historial de comandos hacia abajo</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Panel de análisis</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Atajo</th>
                                <th>Descripción</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>Click</td>
                                <td>Seleccionar/deseleccionar una jugada (mostrar/ocultar flechas)</td>
                            </tr>
                            <tr>
                                <td>Up, k</td>
                                <td>Seleccionar la jugada anterior (cuando hay una jugada seleccionada)</td>
                            </tr>
                            <tr>
                                <td>Down, j</td>
                                <td>Seleccionar la jugada siguiente (cuando hay una jugada seleccionada)</td>
                            </tr>
                            <tr>
                                <td>d</td>
                                <td>Alternar entre análisis de fichas y de cubo (solo en navegación por match)</td>
                            </tr>
                            <tr>
                                <td>Esc</td>
                                <td>Deseleccionar la jugada. Si no hay ninguna jugada seleccionada, cerrar el panel.</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Panel de historial de búsquedas</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Atajo</th>
                                <th>Descripción</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>Click</td>
                                <td>Seleccionar/deseleccionar una búsqueda (mostrar posición)</td>
                            </tr>
                            <tr>
                                <td>Double-click</td>
                                <td>Ejecutar la búsqueda y cerrar el panel</td>
                            </tr>
                            <tr>
                                <td>Up, k</td>
                                <td>Seleccionar la búsqueda anterior (más reciente, arriba)</td>
                            </tr>
                            <tr>
                                <td>Down, j</td>
                                <td>Seleccionar la búsqueda siguiente (más antigua, abajo)</td>
                            </tr>
                            <tr>
                                <td>Esc</td>
                                <td>Deseleccionar la búsqueda. Si no hay ninguna búsqueda seleccionada, cerrar el panel.</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Panel Biblioteca de filtros</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Atajo</th>
                                <th>Descripción</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>Click</td>
                                <td>Seleccionar/deseleccionar un filtro (mostrar posición)</td>
                            </tr>
                            <tr>
                                <td>Double-click</td>
                                <td>Ejecutar la búsqueda del filtro</td>
                            </tr>
                            <tr>
                                <td>Up, k</td>
                                <td>Seleccionar el filtro anterior (cuando hay un filtro seleccionado)</td>
                            </tr>
                            <tr>
                                <td>Down, j</td>
                                <td>Seleccionar el filtro siguiente (cuando hay un filtro seleccionado)</td>
                            </tr>
                            <tr>
                                <td>Esc</td>
                                <td>Deseleccionar el filtro. Si no hay ningún filtro seleccionado, cerrar el panel.</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Panel Match</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Atajo</th>
                                <th>Descripción</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>Click</td>
                                <td>Seleccionar/deseleccionar un match</td>
                            </tr>
                            <tr>
                                <td>Double-click</td>
                                <td>Cargar las posiciones del match</td>
                            </tr>
                            <tr>
                                <td>Up, k</td>
                                <td>Seleccionar el match anterior</td>
                            </tr>
                            <tr>
                                <td>Down, j</td>
                                <td>Seleccionar el match siguiente</td>
                            </tr>
                            <tr>
                                <td>Del</td>
                                <td>Eliminar el match seleccionado</td>
                            </tr>
                            <tr>
                                <td>Esc</td>
                                <td>Deseleccionar el match. Si no hay ningún match seleccionado, cerrar el panel.</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Panel Colección</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Atajo</th>
                                <th>Descripción</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>Click</td>
                                <td>Seleccionar una colección (mostrar sus posiciones)</td>
                            </tr>
                            <tr>
                                <td>Double-click</td>
                                <td>Abrir la colección y recorrer sus posiciones</td>
                            </tr>
                            <tr>
                                <td>Del</td>
                                <td>Quitar la posición actual de la colección activa</td>
                            </tr>
                            <tr>
                                <td>Esc</td>
                                <td>Deseleccionar la colección. Si no hay ninguna colección seleccionada, cerrar el panel.</td>
                            </tr>
                        </tbody>
                    </table>

                    <h3>Panel Anki</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Atajo</th>
                                <th>Descripción</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>1</td>
                                <td>Calificar: Otra vez (repaso fallido, mostrar de nuevo pronto)</td>
                            </tr>
                            <tr>
                                <td>2</td>
                                <td>Calificar: Difícil (recuerdo difícil)</td>
                            </tr>
                            <tr>
                                <td>3</td>
                                <td>Calificar: Bien (recuerdo correcto)</td>
                            </tr>
                            <tr>
                                <td>4</td>
                                <td>Calificar: Fácil (recuerdo sin esfuerzo)</td>
                            </tr>
                            <tr>
                                <td>Esc</td>
                                <td>Detener el repaso y volver a la lista de mazos (se puede reanudar después)</td>
                            </tr>
                        </tbody>
                    </table>
`,
    commands: `
                    <h3>Base de datos</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Comando</th>
                                <th>Descripción</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>new, ne, n</td>
                                <td>Crear una nueva base de datos</td>
                            </tr>
                            <tr>
                                <td>open, op, o</td>
                                <td>Abrir una base de datos existente</td>
                            </tr>
                            <tr>
                                <td>import_db, idb</td>
                                <td>Importar y fusionar otra base de datos</td>
                            </tr>
                            <tr>
                                <td>export_db, edb</td>
                                <td>Exportar la selección actual a una nueva base de datos</td>
                            </tr>
                            <tr>
                                <td>quit, q</td>
                                <td>Salir de blunderDB</td>
                            </tr>
                            <tr>
                                <td>meta</td>
                                <td>Editar metadatos</td>
                            </tr>
                        </tbody>
                    </table>
                    <h3>Posición</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Comando</th>
                                <th>Descripción</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>import, i</td>
                                <td>Importar una posición o match</td>
                            </tr>
                            <tr>
                                <td>write, wr, w</td>
                                <td>Guardar una posición</td>
                            </tr>
                            <tr>
                                <td>write!, wr!, w!</td>
                                <td>Actualizar una posición</td>
                            </tr>
                            <tr>
                                <td>delete, del, d</td>
                                <td>Eliminar una posición</td>
                            </tr>
                        </tbody>
                    </table>
                    <h3>Navegación</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Comando</th>
                                <th>Descripción</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>[número]</td>
                                <td>Ir a una posición concreta por índice</td>
                            </tr>
                        </tbody>
                    </table>
                    <h3>Herramientas</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Comando</th>
                                <th>Descripción</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>list, l</td>
                                <td>Mostrar análisis</td>
                            </tr>
                            <tr>
                                <td>comment, co</td>
                                <td>Escribir comentarios</td>
                            </tr>
                            <tr>
                                <td>filter, fl</td>
                                <td>Mostrar la biblioteca de filtros</td>
                            </tr>
                            <tr>
                                <td>history, hi</td>
                                <td>Mostrar el historial de búsquedas</td>
                            </tr>
                            <tr>
                                <td>match, ma</td>
                                <td>Mostrar el panel Match</td>
                            </tr>
                            <tr>
                                <td>collection, coll</td>
                                <td>Mostrar el panel de colecciones</td>
                            </tr>
                            <tr>
                                <td>epc</td>
                                <td>Calculadora EPC (Effective Pip Count)</td>
                            </tr>
                            <tr>
                                <td>m</td>
                                <td>Navegar por el último match visitado</td>
                            </tr>
                            <tr>
                                <td>help, he, h</td>
                                <td>Abrir ayuda</td>
                            </tr>
                            <tr>
                                <td>met</td>
                                <td>Abrir la tabla de match equity Kazaross-XG2</td>
                            </tr>
                            <tr>
                                <td>tp2</td>
                                <td>Punto de aceptación con cubo a 2 (Live y Last)</td>
                            </tr>
                            <tr>
                                <td>tp2_live</td>
                                <td>Punto de aceptación con cubo a 2 en carreras largas</td>
                            </tr>
                            <tr>
                                <td>tp2_last</td>
                                <td>Punto de aceptación con cubo a 2 en posiciones de última tirada</td>
                            </tr>
                            <tr>
                                <td>tp4</td>
                                <td>Punto de aceptación con cubo a 4 (Live y Last)</td>
                            </tr>
                            <tr>
                                <td>tp4_live</td>
                                <td>Punto de aceptación con cubo a 4 en carreras largas</td>
                            </tr>
                            <tr>
                                <td>tp4_last</td>
                                <td>Punto de aceptación con cubo a 4 en posiciones de última tirada</td>
                            </tr>
                            <tr>
                                <td>gv1</td>
                                <td>Valores de gammon con cubo a 1</td>
                            </tr>
                            <tr>
                                <td>gv2</td>
                                <td>Valores de gammon con cubo a 2</td>
                            </tr>
                            <tr>
                                <td>gv4</td>
                                <td>Valores de gammon con cubo a 4</td>
                            </tr>
                            <tr>
                                <td>#tag1 tag2 ...</td>
                                <td>Etiquetar posición</td>
                            </tr>
                            <tr>
                                <td>s</td>
                                <td>Buscar posiciones con filtros</td>
                            </tr>
                            <tr>
                                <td>ss</td>
                                <td>Buscar en los resultados actuales con filtros</td>
                            </tr>
                            <tr>
                                <td>e</td>
                                <td>Recargar todas las posiciones</td>
                            </tr>
                        </tbody>
                    </table>
                    <h3>Filtros</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Filtro</th>
                                <th>Descripción</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>cube, cub, cu, c</td>
                                <td>Incluir el cubo</td>
                            </tr>
                            <tr>
                                <td>score, sco, sc, s</td>
                                <td>Incluir el marcador</td>
                            </tr>
                            <tr>
                                <td>d</td>
                                <td>Incluir el tipo de decisión</td>
                            </tr>
                            <tr>
                                <td>D</td>
                                <td>Incluir la tirada de dados</td>
                            </tr>
                            <tr>
                                <td>nc</td>
                                <td>Incluir posiciones sin contacto</td>
                            </tr>
                            <tr>
                                <td>M</td>
                                <td>Incluir posiciones reflejadas</td>
                            </tr>
                            <tr>
                                <td>i</td>
                                <td>Solo posiciones importadas por separado, no traídas por la importación de una partida</td>
                            </tr>
                            <tr>
                                <td>x</td>
                                <td
                                    >Excluir las posiciones que contengan <em>cualquier</em> ficha de la estructura dibujada en la pestaña "Except" (p. ej. dibujar fichas en 1, 3 y 5 conserva solo las posiciones que no tengan ninguna de
                                    ellas). Alterna "At least" / "Except" sobre los filtros para dibujar las fichas excluidas en el tablero (mostradas con una marca roja). El recuento por punto no está limitado (3 en un punto
                                    excluye 3 o más allí — un punto hecho sin ficha sobrante), y dos clics rápidos sobre un punto lo marcan como obligatoriamente vacío (una celda con rayado rojo, de cualquier color); un solo clic sobre ese
                                    punto lo desbloquea. En un punto compartido, "Except" prevalece sobre "At least" cuando ambos se contradicen.</td
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
                                <td>Pip count entre x e y</td>
                            </tr>
                            <tr>
                                <td>P>x</td>
                                <td>Pip count absoluto del jugador &gt; x</td>
                            </tr>
                            <tr>
                                <td>P&lt;x</td>
                                <td>Pip count absoluto del jugador &lt; x</td>
                            </tr>
                            <tr>
                                <td>Px,y</td>
                                <td>Pip count absoluto del jugador entre x e y</td>
                            </tr>
                            <tr>
                                <td>e>x</td>
                                <td>Equity &gt; x (en milipuntos)</td>
                            </tr>
                            <tr>
                                <td>e&lt;x</td>
                                <td>Equity &lt; x (en milipuntos)</td>
                            </tr>
                            <tr>
                                <td>ex,y</td>
                                <td>Equity entre x e y (en milipuntos)</td>
                            </tr>
                            <tr>
                                <td>E>x</td>
                                <td>Error de jugada del jugador 1 &gt; x (en milipuntos)</td>
                            </tr>
                            <tr>
                                <td>E&lt;x</td>
                                <td>Error de jugada del jugador 1 &lt; x (en milipuntos)</td>
                            </tr>
                            <tr>
                                <td>Ex,y</td>
                                <td>Error de jugada del jugador 1 entre x e y (en milipuntos)</td>
                            </tr>
                            <tr>
                                <td>w>x</td>
                                <td>Tasa de victoria &gt; x</td>
                            </tr>
                            <tr>
                                <td>w&lt;x</td>
                                <td>Tasa de victoria &lt; x</td>
                            </tr>
                            <tr>
                                <td>wx,y</td>
                                <td>Tasa de victoria entre x e y</td>
                            </tr>
                            <tr>
                                <td>g>x</td>
                                <td>Tasa de gammon &gt; x</td>
                            </tr>
                            <tr>
                                <td>g&lt;x</td>
                                <td>Tasa de gammon &lt; x</td>
                            </tr>
                            <tr>
                                <td>gx,y</td>
                                <td>Tasa de gammon entre x e y</td>
                            </tr>
                            <tr>
                                <td>b>x</td>
                                <td>Tasa de backgammon &gt; x</td>
                            </tr>
                            <tr>
                                <td>b&lt;x</td>
                                <td>Tasa de backgammon &lt; x</td>
                            </tr>
                            <tr>
                                <td>bx,y</td>
                                <td>Tasa de backgammon entre x e y</td>
                            </tr>
                            <tr>
                                <td>W>x</td>
                                <td>Tasa de victoria del oponente &gt; x</td>
                            </tr>
                            <tr>
                                <td>W&lt;x</td>
                                <td>Tasa de victoria del oponente &lt; x</td>
                            </tr>
                            <tr>
                                <td>Wx,y</td>
                                <td>Tasa de victoria del oponente entre x e y</td>
                            </tr>
                            <tr>
                                <td>G>x</td>
                                <td>Tasa de gammon del oponente &gt; x</td>
                            </tr>
                            <tr>
                                <td>G&lt;x</td>
                                <td>Tasa de gammon del oponente &lt; x</td>
                            </tr>
                            <tr>
                                <td>Gx,y</td>
                                <td>Tasa de gammon del oponente entre x e y</td>
                            </tr>
                            <tr>
                                <td>B>x</td>
                                <td>Tasa de backgammon del oponente &gt; x</td>
                            </tr>
                            <tr>
                                <td>B&lt;x</td>
                                <td>Tasa de backgammon del oponente &lt; x</td>
                            </tr>
                            <tr>
                                <td>Bx,y</td>
                                <td>Tasa de backgammon del oponente entre x e y</td>
                            </tr>
                            <tr>
                                <td>o>x</td>
                                <td>Fichas sacadas del jugador &gt; x</td>
                            </tr>
                            <tr>
                                <td>o&lt;x</td>
                                <td>Fichas sacadas del jugador &lt; x</td>
                            </tr>
                            <tr>
                                <td>ox,y</td>
                                <td>Fichas sacadas del jugador entre x e y</td>
                            </tr>
                            <tr>
                                <td>O>x</td>
                                <td>Fichas sacadas del oponente &gt; x</td>
                            </tr>
                            <tr>
                                <td>O&lt;x</td>
                                <td>Fichas sacadas del oponente &lt; x</td>
                            </tr>
                            <tr>
                                <td>Ox,y</td>
                                <td>Fichas sacadas del oponente entre x e y</td>
                            </tr>
                            <tr>
                                <td>k>x</td>
                                <td>Fichas atrasadas del jugador &gt; x</td>
                            </tr>
                            <tr>
                                <td>k&lt;x</td>
                                <td>Fichas atrasadas del jugador &lt; x</td>
                            </tr>
                            <tr>
                                <td>kx,y</td>
                                <td>Fichas atrasadas del jugador entre x e y</td>
                            </tr>
                            <tr>
                                <td>K>x</td>
                                <td>Fichas atrasadas del oponente &gt; x</td>
                            </tr>
                            <tr>
                                <td>K&lt;x</td>
                                <td>Fichas atrasadas del oponente &lt; x</td>
                            </tr>
                            <tr>
                                <td>Kx,y</td>
                                <td>Fichas atrasadas del oponente entre x e y</td>
                            </tr>
                            <tr>
                                <td>z>x</td>
                                <td>Fichas del jugador en la zona &gt; x</td>
                            </tr>
                            <tr>
                                <td>z&lt;x</td>
                                <td>Fichas del jugador en la zona &lt; x</td>
                            </tr>
                            <tr>
                                <td>zx,y</td>
                                <td>Fichas del jugador en la zona entre x e y</td>
                            </tr>
                            <tr>
                                <td>Z>x</td>
                                <td>Fichas del oponente en la zona &gt; x</td>
                            </tr>
                            <tr>
                                <td>Z&lt;x</td>
                                <td>Fichas del oponente en la zona &lt; x</td>
                            </tr>
                            <tr>
                                <td>Zx,y</td>
                                <td>Fichas del oponente en la zona entre x e y</td>
                            </tr>
                            <tr>
                                <td>bo>x</td>
                                <td>Blot del jugador en el outfield &gt; x</td>
                            </tr>
                            <tr>
                                <td>bo&lt;x</td>
                                <td>Blot del jugador en el outfield &lt; x</td>
                            </tr>
                            <tr>
                                <td>box,y</td>
                                <td>Blot del jugador en el outfield entre x e y</td>
                            </tr>
                            <tr>
                                <td>BO>x</td>
                                <td>Blot del oponente en el outfield &gt; x</td>
                            </tr>
                            <tr>
                                <td>BO&lt;x</td>
                                <td>Blot del oponente en el outfield &lt; x</td>
                            </tr>
                            <tr>
                                <td>BOx,y</td>
                                <td>Blot del oponente en el outfield entre x e y</td>
                            </tr>
                            <tr>
                                <td>bj&gt;x</td>
                                <td>Blot Jan del jugador &gt; x</td>
                            </tr>
                            <tr>
                                <td>bj&lt;x</td>
                                <td>Blot Jan del jugador &lt; x</td>
                            </tr>
                            <tr>
                                <td>bjx,y</td>
                                <td>Blot Jan del jugador entre x e y</td>
                            </tr>
                            <tr>
                                <td>BJ&gt;x</td>
                                <td>Blot Jan del oponente &gt; x</td>
                            </tr>
                            <tr>
                                <td>BJ&lt;x</td>
                                <td>Blot Jan del oponente &lt; x</td>
                            </tr>
                            <tr>
                                <td>BJx,y</td>
                                <td>Blot Jan del oponente entre x e y</td>
                            </tr>

                            <tr>
                                <td>t"word1;word2;..."</td>
                                <td>Buscar texto</td>
                            </tr>
                            <tr>
                                <td>m"pattern1;pattern2;..."</td>
                                <td>Mejores jugadas que contengan al menos uno de los patrones indicados</td>
                            </tr>
                            <tr>
                                <td>m"ND;DT;DP;..."</td>
                                <td>Mejor decisión de cubo entre No Double/Take, Double/Take, Double/Pass</td>
                            </tr>
                            <tr>
                                <td>T&gt;x</td>
                                <td>Fecha de creación &gt; x (año/mes/día)</td>
                            </tr>
                            <tr>
                                <td>T&lt;x</td>
                                <td>Fecha de creación &lt; x (año/mes/día)</td>
                            </tr>
                            <tr>
                                <td>Tx,y</td>
                                <td>Fecha de creación entre x e y</td>
                            </tr>
                            <tr>
                                <td>max</td>
                                <td>Buscar en el match con ID x (p. ej. ma3)</td>
                            </tr>
                            <tr>
                                <td>max,y</td>
                                <td>Buscar en los matches con IDs de x a y (p. ej. ma2,5)</td>
                            </tr>
                            <tr>
                                <td>tnx</td>
                                <td>Buscar en el torneo con ID x (p. ej. tn1)</td>
                            </tr>
                            <tr>
                                <td>tnx,y</td>
                                <td>Buscar en los torneos con IDs de x a y (p. ej. tn1,3)</td>
                            </tr>
                        </tbody>
                    </table>
                    <h3>Varios</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Comando</th>
                                <th>Descripción</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>clear, cl</td>
                                <td>Borrar el historial de comandos</td>
                            </tr>
                            <tr>
                                <td>migrate_from_1_0_to_1_1</td>
                                <td>Migrar la base de datos de la versión 1.0 a la 1.1</td>
                            </tr>
                            <tr>
                                <td>migrate_from_1_1_to_1_2</td>
                                <td>Migrar la base de datos de la versión 1.1 a la 1.2</td>
                            </tr>
                            <tr>
                                <td>migrate_from_1_2_to_1_3</td>
                                <td>Migrar la base de datos de la versión 1.2 a la 1.3</td>
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
                        Desarrollé blunderDB inicialmente para mi uso personal, para detectar patrones en mis errores. Pero es muy grato recibir comentarios, sobre todo cuando se han dedicado muchas horas
                        al diseño, la programación, la depuración... Así que no dudes en escribirme para compartir tus comentarios.
                    </p>
                    <p>Aquí tienes varias formas de contactarme:</p>
                    <ul>
                        <li>Habla conmigo si coincidimos en un torneo,</li>
                        <li>Envíame un correo electrónico,</li>
                    </ul>
                    <h3>Licencia</h3>
                    <p>
                        blunderDB se distribuye bajo la licencia MIT. Esto significa que eres libre de usar, copiar, modificar, fusionar, publicar, distribuir, sublicenciar y/o vender copias del software, siempre
                        que el aviso de copyright original y este aviso de permiso se incluyan en todas las copias o partes sustanciales del software.
                    </p>
                    <h3>Agradecimientos</h3>
                    <p>Dedico este pequeño software a mi pareja <strong>Anne-Claire</strong> y a nuestra querida hija <strong>Perrine</strong>. Quiero dar las gracias especialmente a algunos amigos:</p>
                    <ul>
                        <li>
                            <strong>Tristan Remille</strong>, por iniciarme en el backgammon con alegría y amabilidad; por mostrarme el Camino para comprender este maravilloso juego; por seguir
                            apoyándome a pesar de mis pobres intentos de jugar mejor.
                        </li>
                        <li><strong>Nicolas Harmand</strong>, un compañero alegre durante más de una década en grandes aventuras, y un fantástico compañero de juego desde que le picó el gusanillo del backgammon.</li>
                    </ul>
                    <p>La tabla de match equity Kazaross-XG2 (MET) se atribuye a <strong>Neil Kazaross</strong>.</p>
                    <p>Las tablas de puntos de aceptación y de valores de gammon están tomadas del libro <em>The Theory of Backgammon</em> de <strong>Dirk Schiemann</strong>.</p>
                    <p>
                        La base de datos de bear off de un solo lado de 6 puntos utilizada para el cálculo del EPC (Effective Pip Count) se ha generado con <strong>GNU Backgammon</strong> (GNUbg). GNUbg es un software libre y de código abierto
                        de backgammon distribuido bajo la Licencia Pública General GNU.
                    </p>
`
};
