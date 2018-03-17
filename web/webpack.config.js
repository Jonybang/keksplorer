"use strict";

var ExtractTextPlugin = require('extract-text-webpack-plugin');
var extractScss = new ExtractTextPlugin('./public/assets/build/[name]');

var config = {
	name: "Frontend",
	entry: {
		'babel-polyfill': 'babel-polyfill',
		'app.css': './scss/app.scss',
		'app.js': './js/app.js'
	},
	output: {
		filename: './public/assets/build/[name]'
	},
	resolve: {
		modules: ["node_modules"]
	},
	externals: {
		// require("jquery") is external and available
		//  on the global var jQuery
		"jquery": "jQuery"
	},
    devtool: 'inline-source-map',
    module: {
        rules: [
            {
                test: /\.(ttf|woff|woff2|eot)$/,
                loader: 'file-loader',
                options: {
                    name: 'fonts/[name].[ext]'
                }
            },
            {
                test: /\.(css|scss)/,
                use: extractScss.extract({
                        fallback: "style-loader",
                        use: [ {
                            loader: "css-loader"
                        }, {
                            loader: "resolve-url-loader"
                        }, {
                            loader: "sass-loader",
                            options: {autoprefixer: true}
                        }]
                    })
            },
			{
				test: /\.js$/,
				loader: 'babel-loader',
				exclude: /node_modules/
			}
        ]
    },
	plugins: [
		extractScss
	]
};

module.exports = config;