namespace $.$$ {
	export class $trip2g_admin_noteview_graph_cytoscape extends $.$trip2g_admin_noteview_graph_cytoscape {
		static cytoscape(): any {
			return $mol_import.script( 'https://cdnjs.cloudflare.com/ajax/libs/cytoscape/3.32.0/cytoscape.min.js' ).cytoscape
		}

		cytoscape(): any {
			return $trip2g_admin_noteview_graph_cytoscape.cytoscape()
		}

		render() {
			this.cytoscape()( {
				container: this.dom_node(),
				elements: [ // list of graph elements to start with
					{ // node a
						data: { id: 'a' }
					},
					{ // node b
						data: { id: 'b' }
					},
					{ // edge ab
						data: { id: 'ab', source: 'a', target: 'b' }
					}
				],

				style: [ // the stylesheet for the graph
					{
						selector: 'node',
						style: {
							'background-color': '#666',
							'label': 'data(id)'
						}
					},

					{
						selector: 'edge',
						style: {
							'width': 3,
							'line-color': '#ccc',
							'target-arrow-color': '#ccc',
							'target-arrow-shape': 'triangle',
							'curve-style': 'bezier'
						}
					}
				],

				layout: {
					name: 'grid',
					rows: 1
				}
			} )

			return 'test'
		}
	}
}