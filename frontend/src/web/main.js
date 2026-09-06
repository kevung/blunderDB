// Le point d'entrée du front web (#295, fiche J.5).
//
// Il n'y a rien ici : ni magasin global, ni routeur, ni internationalisation.
// C'est le périmètre de l'ADR-0039 rendu visible — une page qui consulte,
// cherche et révise n'a pas besoin d'une architecture, et lui en donner une
// serait le premier pas vers la seconde application que l'ADR refuse.
// Les MÊMES jetons que l'application (ADR-0031, ADR-0008). Le front web n'a
// ni palette ni échelle de taille à lui : deux palettes seraient deux
// produits, et les gardes qui tiennent l'une tiennent alors l'autre.
//
// `tokens.css` et non `style.css` : les jetons seuls, sans les @font-face.
// La page n'affiche pas un caractère japonais et n'a donc aucune raison
// d'embarquer une police japonaise ; elle prend la pile système après
// « Nunito ».
import '../tokens.css';
import { mount } from 'svelte';
import WebApp from './WebApp.svelte';

mount(WebApp, { target: document.getElementById('app') });
