package common

import "strings"

// ClassPresetFor returns the default ClassPreset for the given class name.
// The lookup is case-insensitive. An empty ClassPreset is returned for
// unknown class names.
func ClassPresetFor(className string) ClassPreset {
	switch strings.ToLower(strings.TrimSpace(className)) {
	case "bardo":
		return ClassPreset{
			Traits:    "0 Agilita, -1 Forza, +1 Astuzia, 0 Istinto, +2 Presenza, +1 Conoscenza",
			Primary:   "Stocco - Presenza, Mischia - d8 fis - A una mano",
			Secondary: "Stiletto - Astuzia, Mischia - d8 fis - A una mano",
			Armor:     "Gambesone - Soglie 5/11 - Punteggio Base 3 (Flessibile: +1 all'Evasione)",
			ExtraA:    "un racconto romantico",
			ExtraB:    "una lettera mai aperta",
			Abiti:     []string{"stravaganti", "lussuosi", "vistosi", "di una taglia in piu", "cenciosi", "eleganti", "grezzi"},
			Attitude:  []string{"taverniere", "prestigiatore", "circense", "rockstar", "smargiasso"},
		}
	case "consacrato":
		return ClassPreset{
			Traits:    "0 Agilita, +2 Forza, 0 Astuzia, +1 Istinto, +1 Presenza, -1 Conoscenza",
			Primary:   "Ascia Consacrata - Forza, Mischia - d8+1 mag - A una mano",
			Secondary: "Scudo Rotella - Forza, Mischia - d4 fis - A una mano",
			Armor:     "Cotta di Maglia - Soglie 7/15 - Punteggio Base 4 (Pesante: -1 all'Evasione)",
			ExtraA:    "una raccolta di offerte",
			ExtraB:    "il simbolo sacro della vostra divinita",
			Abiti:     []string{"splendenti", "ondeggianti", "ornati", "aderenti", "modesti", "strani", "naturali"},
			Attitude:  []string{"angelico", "di un medico", "di un predicatore", "monastico", "sacerdotale"},
		}
	case "druido":
		return ClassPreset{
			Traits:    "+1 Agilita, 0 Forza, +1 Astuzia, +2 Istinto, -1 Presenza, 0 Conoscenza",
			Primary:   "Verga - Istinto, Ravvicinata - d8+1 mag - A una mano",
			Secondary: "Scudo Rotella - Forza, Mischia - d4 fis - A una mano",
			Armor:     "Corazza di Cuoio - Soglie 6/13 - Punteggio Base 3",
			ExtraA:    "una borsa piena di pietruzze e ossicini",
			ExtraB:    "uno strano pendente scovato nella sporcizia",
			Abiti:     []string{"mimetici", "di fibre vegetali", "confortevoli", "naturali", "di pezze cucite insieme", "regali", "stracci"},
			Attitude:  []string{"esplosivo", "astuto come una volpe", "da guida nella foresta", "come un figlio dei fiori", "stregonesco"},
		}
	case "fuorilegge":
		return ClassPreset{
			Traits:    "+1 Agilita, -1 Forza, +2 Astuzia, 0 Istinto, +1 Presenza, 0 Conoscenza",
			Primary:   "Pugnale - Astuzia, Mischia - d8+1 fis - A una mano",
			Secondary: "Stiletto - Astuzia, Mischia - d8 fis - A una mano",
			Armor:     "Gambesone - Soglie 5/11 - Punteggio Base 3 (Flessibile: +1 all'Evasione)",
			ExtraA:    "attrezzatura da falsario",
			ExtraB:    "un rampino",
			Abiti:     []string{"puliti", "scuri", "anonimi", "in pelle", "inquietanti", "mimetici", "tattici", "aderenti"},
			Attitude:  []string{"da bandito", "da truffatore", "da giocatore d'azzardo", "da capobanda", "da pirata"},
		}
	case "guardiano":
		return ClassPreset{
			Traits:    "+1 Agilita, +2 Forza, -1 Astuzia, 0 Istinto, +1 Presenza, 0 Conoscenza",
			Primary:   "Ascia da Battaglia - Forza, Mischia - d10+3 fis - A due mani",
			Secondary: "",
			Armor:     "Cotta di Maglia - Soglie 7/15 - Punteggio Base 4 (Pesante: -1 all'Evasione)",
			ExtraA:    "un ricordo del vostro mentore",
			ExtraB:    "una chiave misteriosa",
			Abiti:     []string{"casual", "ornati", "confortevoli", "imbottiti", "regali", "tattici", "consunti"},
			Attitude:  []string{"di un capitano", "di un guardiano", "di un elefante", "di un generale", "di un lottatore"},
		}
	case "guerriero":
		return ClassPreset{
			Traits:    "+2 Agilita, +1 Forza, 0 Astuzia, +1 Istinto, -1 Presenza, 0 Conoscenza",
			Primary:   "Spada Lunga - Agilita, Mischia - d8+3 fis - A due mani",
			Secondary: "",
			Armor:     "Cotta di Maglia - Soglie 7/15 - Punteggio Base 4 (Pesante: -1 all'Evasione)",
			ExtraA:    "il ritratto di chi amate",
			ExtraB:    "una cote per affilare",
			Abiti:     []string{"provocanti", "rattoppati", "rinforzati", "regali", "eleganti", "di ricambio", "consunti"},
			Attitude:  []string{"da toro", "da soldato fedele", "da gladiatore", "eroico", "da mercenario"},
		}
	case "mago":
		return ClassPreset{
			Traits:    "-1 Agilita, 0 Forza, 0 Astuzia, +1 Istinto, +1 Presenza, +2 Conoscenza",
			Primary:   "Bordone - Conoscenza, Remota - d6 mag - A due mani",
			Secondary: "",
			Armor:     "Corazza di Cuoio - Soglie 6/13 - Punteggio Base 3",
			ExtraA:    "un libro che state cercando di tradurre",
			ExtraB:    "un piccolo e innocuo cucciolo elementale",
			Abiti:     []string{"belli", "puliti", "ordinari", "fluenti", "a strati", "rattoppati", "aderenti"},
			Attitude:  []string{"eccentrico", "da bibliotecario", "di una miccia accesa", "da filosofo", "da professore"},
		}
	case "ranger":
		return ClassPreset{
			Traits:    "+2 Agilita, 0 Forza, +1 Astuzia, +1 Istinto, -1 Presenza, 0 Conoscenza",
			Primary:   "Arco Corto - Agilita, Lontana - d6+3 fis - A due mani",
			Secondary: "",
			Armor:     "Corazza di Cuoio - Soglie 6/13 - Punteggio Base 3",
			ExtraA:    "un trofeo della vostra prima preda",
			ExtraB:    "una bussola apparentemente rotta",
			Abiti:     []string{"fluenti", "dai colori spenti", "naturali", "macchiati", "tattici", "aderenti", "di lana o di lino"},
			Attitude:  []string{"di un bambino", "spettrale", "di un survivalista", "di un insegnante", "di un cane da guardia"},
		}
	case "stregone":
		return ClassPreset{
			Traits:    "0 Agilita, -1 Forza, +1 Astuzia, +2 Istinto, +1 Presenza, 0 Conoscenza",
			Primary:   "Bastone Doppio - Istinto, Lontana - 1d6+3 mag - A due mani",
			Secondary: "",
			Armor:     "Gambesone - Soglie 5/11 - Punteggio Base 3 (Flessibile: +1 all'Evasione)",
			ExtraA:    "un globo sussurrante",
			ExtraB:    "un cimelio di famiglia",
			Abiti:     []string{"a strati", "aderenti", "decorati", "poco appariscenti", "sempre in movimento", "sgargianti"},
			Attitude:  []string{"burlone", "da celebrita", "da condottiero", "da politico", "da lupo travestito da agnello"},
		}
	default:
		return ClassPreset{}
	}
}
