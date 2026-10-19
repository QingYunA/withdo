package data

import "context"

// The Recent changes requests of the FakeAssistant.

// recentWait holds a Recent request at RecentBlock, and reports a cancelled one.
func (f *FakeAssistant) recentWait(ctx context.Context) error {
	f.mu.Lock()
	block := f.RecentBlock
	f.mu.Unlock()
	if block != nil {
		select {
		case <-block:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return ctx.Err()
}

// DescribeRecent implements Assistant.
func (f *FakeAssistant) DescribeRecent(ctx context.Context, req RecentDescribeRequest) (map[int]RecentNote, error) {
	f.mu.Lock()
	f.described = append(f.described, req)
	script := f.RecentScript
	f.mu.Unlock()
	if err := f.recentWait(ctx); err != nil {
		return nil, err
	}
	if script != nil {
		return script(req)
	}
	out := map[int]RecentNote{}
	for _, i := range req.Items {
		out[i.Number] = FallbackNote(i)
	}
	return out, nil
}

// WriteRecentLines implements Assistant.
func (f *FakeAssistant) WriteRecentLines(ctx context.Context, req RecentLinesRequest) (string, error) {
	f.mu.Lock()
	f.lined = append(f.lined, req)
	script := f.RecentLinesScript
	f.mu.Unlock()
	if err := f.recentWait(ctx); err != nil {
		return "", err
	}
	if script != nil {
		return script(req)
	}
	return "Some things went in.", nil
}

// ExplainRecent implements Assistant.
func (f *FakeAssistant) ExplainRecent(ctx context.Context, req RecentExplainRequest) (RecentExplanation, error) {
	f.mu.Lock()
	f.explained = append(f.explained, req)
	script := f.RecentExplainScript
	f.mu.Unlock()
	if err := f.recentWait(ctx); err != nil {
		return RecentExplanation{}, err
	}
	if script != nil {
		return script(req)
	}
	return RecentExplanation{What: "It does a thing.", Why: "Somebody asked.", Where: "In the code."}, nil
}

// RecentDescribed is every request for the sentences of a repository's items, in order.
func (f *FakeAssistant) RecentDescribed() []RecentDescribeRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]RecentDescribeRequest(nil), f.described...)
}

// RecentLined is every request for a repository's lines, in order.
func (f *FakeAssistant) RecentLined() []RecentLinesRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]RecentLinesRequest(nil), f.lined...)
}

// RecentExplained is every request to explain an item, in order.
func (f *FakeAssistant) RecentExplained() []RecentExplainRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]RecentExplainRequest(nil), f.explained...)
}
