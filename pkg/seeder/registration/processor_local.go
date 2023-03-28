package registration

import "context"

func (p *Processor) getRequestLocally(_ context.Context, req *Request) (*cert, bool) {
	p.certsCacheLock.RLock()
	defer p.certsCacheLock.RUnlock()

	certTmp, ok := p.certsCache[req.DeviceID]
	if !ok {
		return nil, false
	}

	cert := *certTmp
	cert.der = make([]byte, len(certTmp.der))
	copy(cert.der, certTmp.der)
	return &cert, true
}

func (p *Processor) addRequestLocally(_ context.Context, req *Request) {
	// make an entry in the cache immediately
	p.certsCacheLock.Lock()
	p.certsCache[req.DeviceID] = &cert{}
	p.certsCacheLock.Unlock()
}

func (p *Processor) deleteRequestLocally(_ context.Context, req *Request) {
	p.certsCacheLock.Lock()
	delete(p.certsCache, req.DeviceID)
	p.certsCacheLock.Unlock()
}

func (p *Processor) processRequestLocally(req *Request) {

}
