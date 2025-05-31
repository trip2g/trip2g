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
			const nodeFilter = (item: any) => !item.id.includes('sidebar')
			const data = this.data().filter(nodeFilter).map(item => ({
				...item,
				inLinks: item.inLinks.filter(nodeFilter),
			}))

			// Clear all existing elements from the graph
			cy.elements().remove()

			// Prepare nodes and edges from the data
			const elements = []
			const nodeIds = new Set()

			// Add all nodes first
			for (const node of data) {
				if (!nodeIds.has(node.id)) {
					elements.push({
						data: { 
							id: node.id,
							label: node.id 
						}
					})
					nodeIds.add(node.id)
				}

				// Add linked nodes
				for (const inLink of node.inLinks) {
					if (!nodeIds.has(inLink.id)) {
						elements.push({
							data: { 
								id: inLink.id,
								label: inLink.title 
							}
						})
						nodeIds.add(inLink.id)
					}
				}
			}

			// Add edges
			for (const node of data) {
				for (const inLink of node.inLinks) {
					elements.push({
						data: { 
							id: `${inLink.id}-${node.id}`,
							source: inLink.id,
							target: node.id 
						}
					})
				}
			}

			// Add elements to cytoscape
			cy.add(elements)

			// Apply styling
			cy.style([
				{
					selector: 'node',
					style: {
						'background-color': '#666',
						'label': 'data(label)',
						'text-valign': 'center',
						'text-halign': 'center',
						'font-size': '12px',
						'color': '#fff'
					}
				},
				{
					selector: 'edge',
					style: {
						'width': 2,
						'line-color': '#ccc',
						'target-arrow-color': '#ccc',
						'target-arrow-shape': 'triangle',
						'curve-style': 'bezier'
					}
				}
			])

			// Apply layout
			cy.layout({
				name: 'cose',
				idealEdgeLength: 100,
				nodeOverlap: 20,
				refresh: 20,
				fit: true,
				padding: 30,
				randomize: false,
				componentSpacing: 100,
				nodeRepulsion: 400000,
				edgeElasticity: 100,
				nestingFactor: 5,
				gravity: 80,
				numIter: 1000,
				initialTemp: 200,
				coolingFactor: 0.95,
				minTemp: 1.0
			}).run()
		}
	}
}