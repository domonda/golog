package golog

// // MultiWriter is a Writer that writes to multiple other Writers
// type MultiWriter []Writer

// func joinWriters(a Writer, b ...Writer) Writer {
// 	if len(b) == 0 {
// 		return uniqueWriters(a, nil)
// 	}
// 	return uniqueWriters(append(MultiWriter{a}, b...), nil)
// }

// func uniqueWriters(w Writer, exclude []Writer) Writer {
// 	if w == nil || slices.Contains(exclude, w) {
// 		return nil
// 	}
// 	mw, ok := w.(MultiWriter)
// 	if !ok {
// 		return w
// 	}
// 	if len(mw) == 0 {
// 		return nil
// 	}
// 	numUnique := 0
// 	isFlat := true
// 	for i, w := range mw {
// 		// Count as unique if the Writer in the MultiWriter is not nil,
// 		// unique in the itself MultiWriter and not in the exclude list
// 		if w != nil && !slices.Contains(mw[:i], w) && !slices.Contains(exclude, w) {
// 			numUnique++
// 		}
// 		if _, ok = w.(MultiWriter); ok {
// 			isFlat = false
// 		}
// 	}
// 	if numUnique == len(mw) && isFlat {
// 		return w
// 	}
// 	unique := make(MultiWriter, 0, numUnique)
// 	for i, w := range mw {
// 		if w == nil || slices.Contains(mw[:i], w) || slices.Contains(exclude, w) {
// 			continue
// 		}
// 		if _, ok = w.(MultiWriter); !ok {
// 			unique = append(unique, w)
// 		}
// 		// w is a MultiWriter, recursively extracte unique Writers
// 		uw := uniqueWriters(w, append(exclude, unique...))
// 		// uniqueWriters can return nil or a single non MultiWriter Writer
// 		if uw == nil {
// 			continue
// 		}
// 		if uwmw, ok := uw.(MultiWriter); ok {
// 			unique = append(unique, uwmw...)
// 		} else {
// 			unique = append(unique, uw)
// 		}
// 	}
// 	switch len(unique) {
// 	case 0:
// 		return nil
// 	case 1:
// 		return unique[0]
// 	}
// 	return unique
// }

// var multiWriterPool sync.Pool

// func getMultiWriter(numWriters int) MultiWriter {
// 	if recycled, ok := multiWriterPool.Get().(MultiWriter); ok && numWriters <= cap(recycled) {
// 		return recycled[:numWriters]
// 	}
// 	return make(MultiWriter, numWriters)
// }

// func (m MultiWriter) BeginMessage(ctx context.Context, logger *Logger, t time.Time, level Level, text string) Writer {
// 	next := getMultiWriter(len(m))
// 	for i, w := range m {
// 		next[i] = w.BeginMessage(ctx, logger, t, level, text)
// 	}
// 	return next
// }

// func (m MultiWriter) CommitMessage() {
// 	for i, f := range m {
// 		f.CommitMessage()
// 		m[i] = nil
// 	}
// 	multiWriterPool.Put(m)
// }

// func (m MultiWriter) FlushUnderlying() {
// 	for _, f := range m {
// 		if f != nil {
// 			f.FlushUnderlying()
// 		}
// 	}
// }

// func (m MultiWriter) String() string {
// 	var b strings.Builder
// 	for i, f := range m {
// 		if i > 0 {
// 			b.WriteByte('\n')
// 		}
// 		b.WriteString(f.String())
// 	}
// 	return b.String()
// }

// func (m MultiWriter) WriteKey(key string) {
// 	for _, f := range m {
// 		f.WriteKey(key)
// 	}
// }

// func (m MultiWriter) WriteSliceKey(key string) {
// 	for _, f := range m {
// 		f.WriteSliceKey(key)
// 	}
// }

// func (m MultiWriter) WriteSliceEnd() {
// 	for _, f := range m {
// 		f.WriteSliceEnd()
// 	}
// }

// func (m MultiWriter) WriteNil() {
// 	for _, f := range m {
// 		f.WriteNil()
// 	}
// }

// func (m MultiWriter) WriteBool(val bool) {
// 	for _, f := range m {
// 		f.WriteBool(val)
// 	}
// }

// func (m MultiWriter) WriteInt(val int64) {
// 	for _, f := range m {
// 		f.WriteInt(val)
// 	}
// }

// func (m MultiWriter) WriteUint(val uint64) {
// 	for _, f := range m {
// 		f.WriteUint(val)
// 	}
// }

// func (m MultiWriter) WriteFloat(val float64) {
// 	for _, f := range m {
// 		f.WriteFloat(val)
// 	}
// }

// func (m MultiWriter) WriteString(val string) {
// 	for _, f := range m {
// 		f.WriteString(val)
// 	}
// }

// func (m MultiWriter) WriteError(val error) {
// 	for _, f := range m {
// 		f.WriteError(val)
// 	}
// }

// func (m MultiWriter) WriteUUID(val [16]byte) {
// 	for _, f := range m {
// 		f.WriteUUID(val)
// 	}
// }

// func (m MultiWriter) WriteJSON(val []byte) {
// 	for _, f := range m {
// 		f.WriteJSON(val)
// 	}
// }
