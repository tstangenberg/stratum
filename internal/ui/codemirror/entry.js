import { EditorView } from '@codemirror/view';
import { EditorState } from '@codemirror/state';
import { basicSetup } from 'codemirror';
import { javascript } from '@codemirror/lang-javascript';
import { graphqlLanguageSupport } from 'cm6-graphql';
import { setDiagnostics } from '@codemirror/lint';

window.CodeMirror6 = { EditorView, EditorState, basicSetup, javascript, graphql: graphqlLanguageSupport, setDiagnostics };
