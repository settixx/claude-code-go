package buddy

// SpriteSet holds multi-frame ASCII art for each mood state.
type SpriteSet struct {
	Idle     []string
	Happy    []string
	Thinking []string
	Sleeping []string
	Excited  []string
}

func GetSprites(species string) SpriteSet {
	switch species {
	case "cat":
		return catSprites()
	case "ghost":
		return ghostSprites()
	case "robot":
		return robotSprites()
	case "bear":
		return bearSprites()
	default:
		return duckSprites()
	}
}

func duckSprites() SpriteSet {
	return SpriteSet{
		Idle: []string{
			"  __\n (o>\n /| \n / |",
			"  __\n (o>\n /|\\\n / |",
		},
		Happy: []string{
			"  __\n (^>\n\\|/\n / |",
			"  __\n (^>\n /|\\\n / |",
		},
		Thinking: []string{
			"  __  ?\n (o>\n /| \n / |",
			"  __ ??\n (o>\n /| \n / |",
		},
		Sleeping: []string{
			"  __ z\n (->\n /| \n / |",
			"  __ zZ\n (->\n /| \n / |",
			"  __ zZz\n (->\n /| \n / |",
		},
		Excited: []string{
			"  __  !\n (O>\n\\|/\n / |",
			"  __  !\n (O> ~\n /|\\\n / |",
		},
	}
}

func catSprites() SpriteSet {
	return SpriteSet{
		Idle: []string{
			" /\\_/\\\n( o.o )\n > ^ <",
			" /\\_/\\\n( o.o )\n > v <",
		},
		Happy: []string{
			" /\\_/\\\n( ^.^ )\n > ^ < ~",
			" /\\_/\\\n( ^.^ )\n > ^ <~",
		},
		Thinking: []string{
			" /\\_/\\  ?\n( o.o )\n > - <",
			" /\\_/\\ ??\n( o.o )\n > - <",
		},
		Sleeping: []string{
			" /\\_/\\ z\n( -.- )\n > ^ <",
			" /\\_/\\ zZ\n( -.- )\n > ^ <",
		},
		Excited: []string{
			" /\\_/\\  !\n( O.O )\n > w <",
			" /\\_/\\ !!\n( O.O )~\n > w <",
		},
	}
}

func ghostSprites() SpriteSet {
	return SpriteSet{
		Idle: []string{
			" .--.\n( oo )\n |  |\n /\\/\\",
			" .--.\n( oo )\n |  |\n \\/\\/",
		},
		Happy: []string{
			" .--.\n( ^^ )\n |  |\n /\\/\\",
			" .--.\n( ^^ )~\n |  |\n \\/\\/",
		},
		Thinking: []string{
			" .--.  ?\n( oo )\n | .|\n /\\/\\",
			" .--. ??\n( oo )\n |. |\n /\\/\\",
		},
		Sleeping: []string{
			" .--. z\n( -- )\n |  |\n /\\/\\",
			" .--. zZ\n( -- )\n |  |\n /\\/\\",
		},
		Excited: []string{
			" .--. !\n( OO )\n |  |~\n /\\/\\",
			" .--. !\n( OO )~\n | \\|\n \\/\\/",
		},
	}
}

func robotSprites() SpriteSet {
	return SpriteSet{
		Idle: []string{
			" [==]\n |oo|\n /||\\",
			" [==]\n |oo|\n \\||/",
		},
		Happy: []string{
			" [==]\n |^^|\n /||\\ +",
			" [==]\n |^^|\n \\||/ +",
		},
		Thinking: []string{
			" [==] ?\n |oo|\n /||\\",
			" [==]??\n |.o|\n /||\\",
		},
		Sleeping: []string{
			" [==] z\n |--|\n /||\\",
			" [==] zZ\n |--|\n /||\\",
		},
		Excited: []string{
			" [==] !\n |OO|\n/||\\\\",
			" [==] !\n |OO|~\n\\||//",
		},
	}
}

func bearSprites() SpriteSet {
	return SpriteSet{
		Idle: []string{
			"ʕ·ᴥ·ʔ",
			"ʕ·ᴥ· ʔ",
		},
		Happy: []string{
			"ʕ^ᴥ^ʔ",
			"ʕ^ᴥ^ʔノ",
		},
		Thinking: []string{
			"ʕ·ᴥ·ʔ ?",
			"ʕ·ᴥ·ʔ ??",
		},
		Sleeping: []string{
			"ʕ-ᴥ-ʔ z",
			"ʕ-ᴥ-ʔ zZ",
			"ʕ-ᴥ-ʔ zZz",
		},
		Excited: []string{
			"ʕ◉ᴥ◉ʔ !",
			"ʕ◉ᴥ◉ʔ !!",
		},
	}
}
