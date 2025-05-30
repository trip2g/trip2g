namespace $.$$ {
	export class $trip2g_admin_noteview_graph_cytoscape extends $.$trip2g_admin_noteview_graph_cytoscape {
		static cytoscape(): any {
			return $mol_import.script( 'https://cdnjs.cloudflare.com/ajax/libs/cytoscape/3.32.0/cytoscape.min.js' ).cytoscape
		}

		cytoscape(): any {
			return $trip2g_admin_noteview_graph_cytoscape.cytoscape()
		}

		@$mol_mem
		data() {
			const res = $trip2g_graphql_request( `
				query AdminGraph {
					admin {
						allLatestNoteViews {
							nodes {
								id
								inLinks {
									title
									id
								}
							}
						}
					}
				}
			`)

			return res.admin.allLatestNoteViews.nodes
		}

		@$mol_mem
		cytoscape_instance() {
			return this.cytoscape()( {
				container: this.dom_node()
			} )
		}

		render() {
			const cy = this.cytoscape_instance()
			const data = this.data()

			// this.cytoscape()( {
			// 	container: this.dom_node(),
			// 	elements: [ // list of graph elements to start with
			// 		{ // node a
			// 			data: { id: 'a' }
			// 		},
			// 		{ // node b
			// 			data: { id: 'b' }
			// 		},
			// 		{ // edge ab
			// 			data: { id: 'ab', source: 'a', target: 'b' }
			// 		}
			// 	],

			// 	style: [ // the stylesheet for the graph
			// 		{
			// 			selector: 'node',
			// 			style: {
			// 				'background-color': '#666',
			// 				'label': 'data(id)'
			// 			}
			// 		},

			// 		{
			// 			selector: 'edge',
			// 			style: {
			// 				'width': 3,
			// 				'line-color': '#ccc',
			// 				'target-arrow-color': '#ccc',
			// 				'target-arrow-shape': 'triangle',
			// 				'curve-style': 'bezier'
			// 			}
			// 		}
			// 	],

			// 	layout: {
			// 		name: 'grid',
			// 		rows: 1
			// 	}
			// } )

			return 'test'
		}
	}
}