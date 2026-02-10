package messaging

type SolaceMessage struct {
	msgID   string
	body    string
	msgType string
}

func (m SolaceMessage) ID() string   { return m.msgID }
func (m SolaceMessage) Body() string { return m.body }
func (m SolaceMessage) Type() string { return m.msgType }
