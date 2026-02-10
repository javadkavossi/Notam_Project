package messaging

// AllowedFIRs لیست FIRهای مجاز برای فیلتر و تنظیمات اعلان (شامل خاورمیانه، آسیا، آفریقا و اروپا)
var AllowedFIRs = []string{
	// خاورمیانه و آسیا و آفریقا (موجود)
	"OIIX", "LTTT", "ORBB", "OAKX", "OPKR", "UTAV", "UBBB", "UDDD", "UGGG", "OKAC", "OEJD", "OBBB", "OMAE", "OOMM", "OJAC", "OSDI", "OLBB", "UUWV", "UWWW", "ZBBB", "VHHH", "ZBAA", "ZSPD", "VECC", "VOMM", "VABB",
	// اروپا – شمال (UK, IE, NL, BE, DE, DK, NO, SE, FI, Baltic, PL)
	"EBBU", "EDGG", "EDMM", "EDUU", "EDWW", "EDYY", "EETT", "EFIN", "EGGX", "EGPX", "EGTT", "EHAA", "EISN", "EKDK", "ENOB", "ENOR", "EPWW", "ESAA", "ESMM", "EVRR", "EYVL",
	// اروپا – ایسلند و جزایر قناری
	"BIRD", "GCCC",
	// اروپا – فرانسه، اسپانیا، ایتالیا، اتریش، سوئیس، پرتغال، یونان، بالکان، قبرس، مالت، ترکیه (بخش اروپایی)
	"LFFF", "LFEE", "LFBB", "LFMN", "LFRR", "LECM", "LECB", "LIRR", "LIMM", "LIPX", "LOWW", "LOWS", "LSAS", "LPPC", "LPFR", "LGGG", "LGAV", "LDZO", "LDSP", "LJLA", "LQSA", "LBSR", "LBWN", "LRBB", "LHCC", "LKAA", "LZBB", "LCCC", "LMMM", "LTBB", "LAHR", "LWSS", "LYBE", "LYTV",
	// اروپای شرقی
	"UMMV", "UKLV", "URKK", "ULLL", "LUKK",
}

// AllowedAirports لیست کدهای ICAO فرودگاه‌های مجاز (شامل فرودگاه‌های مسافربری اروپا و موجود)
var AllowedAirports = []string{
	// موجود (خاورمیانه، آسیا، آفریقا)
	"OIII", "OIIE", "OISS", "OIMM", "OIMB", "OICC", "OITT", "OIKK", "OIZH", "OIAW", "OIGG", "OIAA", "OIBP", "OIKB", "OIBK", "OIKQ", "OICS", "OIAR", "OIMN", "OIZB", "OIKY", "OISL", "OIFM", "OIFS", "OIKH", "OIKM", "OIOM", "OISR", "OIBL", "OIHR",
	"LTBA", "LTFM", "LTAI", "LTBS", "LTFJ", "LTAC", "LTAF", "LTCI", "LTFE", "LTAU", "LTCP", "LTCG", "LTFH", "LTBU", "LTCK",
	"ORBI", "ORMM", "ORER", "ORSU", "ORNI", "ORSH", "ORTL", "ORAT", "ORBD", "ORQW", "OAKB", "OAMS", "OASD", "OAHN", "OAFZ", "OASG", "OAUZ", "OAZJ",
	"OPKC", "OPRN", "OPLA", "OPPS", "OPQT", "OPFA", "OPBG", "OPMG", "OPSR", "OPKD", "OPJA", "OPST", "OPCH", "OPNH", "OPKL",
	"UTAA", "UTAM", "UTDD", "UTAK", "UTSB", "UTST", "UTNR", "UTAV", "UBBB", "UBBG", "UBBN", "UBBQ", "UBBY", "UBBZ", "UDYZ", "UDLS", "UDSG", "UDYE",
	"UGTB", "UGKO", "UGSS", "UGSB", "UGSA", "UG23", "OKBK", "OKAS", "OKAJ", "OEJN", "OERK", "OEDF", "OEMA", "OETF", "OERF", "OEGN", "OEAH", "OETB", "OESH", "OEAB", "OEBH", "OTHH", "OTBD", "OTBH",
	"OMAA", "OMDB", "OMSJ", "OMFJ", "OMRK", "OMDW", "OMAJ", "OMAL", "OMAM", "OOMS", "OOSA", "OOBR", "OOMA", "OOSH", "OOSN", "OOKB", "OBBI", "OBBS", "OB18", "OJAI", "OJAQ", "OJAM", "OJMF", "OSDI", "OSAP", "OSLK", "OS65", "OS58", "OLBA", "OLKA",
	"ZGGG", "ZSPD", "ZBAA", "ZSSS", "ZUCK", "ZYTX", "ZGSZ", "ZSHC", "ZWWW", "ZJHK", "ZYHB", "ZUUU", "ZPPP", "ZLXY", "ZSNJ", "UUEE", "UUWW", "ULLI", "USSS", "UHPP", "UHHH", "UNKL", "URSS", "UWUU", "UUDD",
	"VABB", "VIDP", "VOMM", "VECC", "VEBI", "VAGO", "VOCI", "VIJP", "VOBL", "VAAH", "VOGO", "VOMD",
	// اروپا – بریتانیا و ایرلند
	"EGLL", "EGKK", "EGCC", "EGPF", "EGPH", "EGGW", "EGLC", "EGNX", "EGNT", "EGGD", "EGBB", "EGNM", "EGAA", "EGAE", "EGPD", "EGSH", "EICK", "EIDW", "EIWF",
	// اروپا – آلمان
	"EDDF", "EDDM", "EDDH", "EDDK", "EDDL", "EDDB", "EDDV", "EDDP", "EDDN", "EDDW", "EDDG", "EDFM", "EDLP", "EDLW",
	// اروپا – فرانسه
	"LFPG", "LFLL", "LFML", "LFBO", "LFMN", "LFPO", "LFBD", "LFRS", "LFST", "LFLS", "LFMK", "LFAT", "LFBI", "LFBZ", "LFKC", "LFOT", "LFRN", "LFSN", "LFSB", "LFMP", "LFTH",
	// اروپا – اسپانیا و جزایر قناری
	"LEMD", "LEBL", "LEMG", "LEAL", "LEPA", "LEGR", "LECU", "LEVC", "LEZL", "LEIB", "GCLP", "GCRR", "GCXO",
	// اروپا – ایتالیا
	"LIRF", "LIML", "LIPZ", "LIME", "LICC", "LIPR", "LIMC", "LIEE", "LICA", "LIPQ", "LIPX", "LIRN", "LIRP", "LIPB", "LIPH", "LIPE", "LIRA", "LICT", "LIBC",
	// اروپا – هلند، بلژیک، لوکزامبورگ
	"EHAM", "EHEH", "EHBK", "EBBR", "EBCI", "ELLX",
	// اروپا – اتریش، سوئیس
	"LOWW", "LOWS", "LOXL", "LOWG", "LOWK", "LSZH", "LSGG", "LSZR", "LSZB", "LSZS",
	// اروپا – پرتغال و مادیرا و آزور
	"LPPT", "LPFR", "LPMA", "LPPD", "LPPR",
	// اروپا – یونان
	"LGAV", "LGRP", "LGSA", "LGMK", "LGKL", "LGHI", "LGIR", "LGKV", "LGSK", "LGTS", "LGTF",
	// اروپا – اسکاندیناوی و بالتیک
	"EKCH", "EKYT", "EKAH", "EKBI", "ENGM", "ENBR", "ENVA", "ENTC", "ESSA", "ESGG", "ESKN", "ESPA", "EFHK", "EFRO", "EFOU", "EETN", "EVRA", "EVVA", "EYVI", "EYKA",
	// اروپا – لهستان، چک، اسلواکی، مجارستان
	"EPWA", "EPKK", "EPGD", "EPWR", "EPPO", "EPBY", "LKPR", "LKTB", "LKKU", "LZIB", "LZTT", "LZKZ", "LHBP", "LHUD", "LHDC",
	// اروپا – رومانی، بلغارستان
	"LROP", "LRBS", "LRCL", "LRIA", "LBSF", "LBWN", "LBPD",
	// اروپا – کرواسی، اسلوونی، بوسنی، صربستان، مونته‌نگرو، مقدونیه، آلبانی
	"LDZA", "LDSP", "LDRI", "LDOS", "LJLA", "LJLJ", "LQSA", "LQTZ", "LYBE", "LYNI", "LYPG", "LYTV", "LWSS", "LWSK", "LAHR", "LATI",
	// اروپا – قبرس، مالت
	"LCLK", "LCPH", "LCRC", "LMML",
	// اروپا – ترکیه (بخش اروپایی و اصلی مسافربری)
	"LTFJ", "LTAC", "LTAI", "LTBS", "LTFE", "LTAF",
	// اروپا – اوکراین، بلاروس، مولداوی، روسیه (بخش اروپایی)
	"UKKK", "UKLL", "UKDR", "UKHP", "UKOH", "UMMS", "UMMG", "LUKK", "ULLI", "UUEE", "URSS", "URKK", "UWGG", "USRR",
	// اروپا – ایسلند، جزایر فارو، گرینلند
	"BIKF", "BIRK", "EKVG", "BGGH",
}

// AllowedFIRsMap برای چک سریع در consumer
func AllowedFIRsMap() map[string]bool {
	m := make(map[string]bool, len(AllowedFIRs))
	for _, f := range AllowedFIRs {
		m[f] = true
	}
	return m
}

// AllowedAirportsMap برای چک سریع در consumer
func AllowedAirportsMap() map[string]bool {
	m := make(map[string]bool, len(AllowedAirports))
	for _, a := range AllowedAirports {
		m[a] = true
	}
	return m
}
